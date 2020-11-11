package mtproto

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"

	mtcrypto "github.com/xelaj/mtproto/crypto"
	ige "github.com/xelaj/mtproto/crypto/aes_ige"
	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/keys"
	"github.com/xelaj/mtproto/service"
)

func handshake(conn net.Conn, publicKey *rsa.PublicKey) (*SessionCredentials, error) {
	nonceFirst, err := randomInt128()
	if err != nil {
		return nil, err
	}

	pqParams := new(service.ResPQ)
	if err := sendUnencrypted(conn, &ReqPQParams{nonceFirst}, pqParams); err != nil {
		return nil, err
	}

	if nonceFirst.Cmp(pqParams.Nonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: wrong nonce")
	}

	found := false
	for _, b := range pqParams.Fingerprints {
		fgpr, err := keys.RSAFingerprint(publicKey)
		if err != nil {
			return nil, err
		}

		if uint64(b) == binary.LittleEndian.Uint64(fgpr) {
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("handshake: can't find fingerprint")
	}

	// (encoding) p_q_inner_data
	pq := big.NewInt(0).SetBytes(pqParams.Pq)
	p, q := splitPQ(pq)
	nonceSecond, err := randomInt256()
	if err != nil {
		return nil, err
	}

	nonceServer := pqParams.ServerNonce
	message, err := tl.Encode(&service.PQInnerData{
		Pq:          pqParams.Pq,
		P:           p.Bytes(),
		Q:           q.Bytes(),
		Nonce:       nonceFirst,
		ServerNonce: nonceServer,
		NewNonce:    nonceSecond,
	})
	if err != nil {
		return nil, err
	}

	hashAndMsg := make([]byte, 255)

	copy(hashAndMsg, append(mtcrypto.Sha1Bytes(message), message...))

	fgpr, err := keys.RSAFingerprint(publicKey)
	if err != nil {
		return nil, err
	}

	var dhResponse service.ServerDHParams
	encryptedData, err := doRSAencrypt(hashAndMsg, publicKey)
	if err != nil {
		return nil, err
	}

	if err := sendUnencrypted(conn, &ReqDHParamsParams{
		Nonce:                nonceFirst,
		ServerNonce:          nonceServer,
		P:                    p.Bytes(),
		Q:                    q.Bytes(),
		PublicKeyFingerprint: int64(binary.LittleEndian.Uint64(fgpr)),
		EncryptedData:        encryptedData,
	}, &dhResponse); err != nil {
		return nil, err
	}

	dhParams, ok := dhResponse.(*service.ServerDHParamsOk)
	if !ok {
		return nil, fmt.Errorf("need *service.ServerDHParamsOk, got: %T", dhResponse)
	}

	if nonceFirst.Cmp(dhParams.Nonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: Wrong nonce")
	}

	if nonceServer.Cmp(dhParams.ServerNonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: Wrong server_nonce")
	}

	// проверку по хешу, удаление рандомных байт происходит в этой функции
	decodedMessage, err := ige.DecryptMessageWithTempKeys(dhParams.EncryptedAnswer, nonceSecond.Int, nonceServer.Int)
	if err != nil {
		return nil, err
	}

	dhi := new(service.ServerDHInnerData)
	if err := tl.Decode(decodedMessage, dhi); err != nil {
		return nil, err
	}

	if nonceFirst.Cmp(dhi.Nonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: wrong nonce")
	}
	if nonceServer.Cmp(dhi.ServerNonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: wrong server_nonce")
	}

	// вот это видимо как раз и есть часть диффи хеллмана, поэтому просто оставим как есть надеюсь сработает
	_, g_b, g_ab := makeGAB(dhi.G, big.NewInt(0).SetBytes(dhi.GA), big.NewInt(0).SetBytes(dhi.DhPrime))

	authKey := g_ab.Bytes()
	if authKey[0] == 0 {
		authKey = authKey[1:]
	}

	authKeyHash := mtcrypto.AuthKeyHash(authKey)

	// что это я пока не знаю, видимо какой то очень специфичный способ сгенерить ключи
	t4 := make([]byte, 32+1+8)
	copy(t4[0:], nonceSecond.Bytes())
	t4[32] = 1
	copy(t4[33:], mtcrypto.Sha1Bytes(authKey)[0:8])
	nonceHash1 := mtcrypto.Sha1Bytes(t4)[4:20]
	salt := make([]byte, tl.LongLen)
	copy(salt, nonceSecond.Bytes()[:8])
	xor(salt, nonceServer.Bytes()[:8])

	serverSalt := int64(binary.LittleEndian.Uint64(salt))

	// (encoding) client_DH_inner_data
	clientDHDataMsg, err := tl.Encode(&service.ClientDHInnerData{
		Nonce:       nonceFirst,
		ServerNonce: nonceServer,
		Retry:       0,
		GB:          g_b.Bytes(),
	})
	if err != nil {
		return nil, err
	}

	var dhGenStatus service.SetClientDHParamsAnswer
	{
		encData, err := ige.EncryptMessageWithTempKeys(clientDHDataMsg, nonceSecond.Int, nonceServer.Int)
		if err != nil {
			return nil, err
		}

		if err := sendUnencrypted(conn, &SetClientDHParamsParams{
			Nonce:         nonceFirst,
			ServerNonce:   nonceServer,
			EncryptedData: encData,
		}, &dhGenStatus); err != nil {
			return nil, fmt.Errorf("sending clientDHParams: %w", err)

		}
	}

	dhg, ok := dhGenStatus.(*service.DHGenOk)
	if !ok {
		return nil, fmt.Errorf("handshake: need DHGenOk")
	}

	if nonceFirst.Cmp(dhg.Nonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: wrong nonce: %v, %v", nonceFirst, dhg.Nonce)
	}

	if nonceServer.Cmp(dhg.ServerNonce.Int) != 0 {
		return nil, fmt.Errorf("handshake: wrong server_nonce: %v, %v", nonceServer, dhg.ServerNonce)
	}

	if !bytes.Equal(nonceHash1, dhg.NewNonceHash1.Bytes()) {
		return nil, fmt.Errorf(
			"handshake: wrong new_nonce_hash1: %v, %v",
			hex.EncodeToString(nonceHash1),
			hex.EncodeToString(dhg.NewNonceHash1.Bytes()),
		)
	}

	return &SessionCredentials{
		AuthKey:     authKey,
		AuthKeyHash: authKeyHash,
		ServerSalt:  serverSalt,
	}, nil
}

func sendUnencrypted(conn net.Conn, request tl.Object, resp interface{}) error {
	msg, err := tl.Encode(request)
	if err != nil {
		return err
	}

	data, err := (&service.UnencryptedMessage{
		Msg:   msg,
		MsgID: generateMessageID(), // он тут нужен?
	}).Serialize()
	if err != nil {
		return err
	}

	size := make([]byte, 4)
	binary.LittleEndian.PutUint32(size, uint32(len(data)))
	_, err = conn.Write(size)
	if err != nil {
		return fmt.Errorf("sending data: %w", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}

	return readUnencrypted(conn, resp)
}

func readUnencrypted(conn net.Conn, response interface{}) error {
	sizeInBytes := make([]byte, 4)
	n, err := conn.Read(sizeInBytes)
	if err != nil {
		return fmt.Errorf("reading message length: %w", err)
	}

	if n != 4 {
		return fmt.Errorf("size is not length of int32, expected 4 bytes, got %d", n)
	}

	data := make([]byte, int(binary.LittleEndian.Uint32(sizeInBytes)))
	if _, err := conn.Read(data); err != nil {
		return err
	}

	unmsg, err := service.DeserializeUnencryptedMessage(data)
	if err != nil {
		return err
	}

	return tl.Decode(unmsg.Msg, response)
}

func randomInt128() (*service.Int128, error) {
	i := &service.Int128{big.NewInt(0)}
	b, err := mtcrypto.RandomBytes(service.Int128Len)
	if err != nil {
		return nil, err
	}

	i.SetBytes(b)
	return i, nil
}

func randomInt256() (*service.Int256, error) {
	i := &service.Int256{big.NewInt(0)}
	b, err := mtcrypto.RandomBytes(service.Int256Len)
	if err != nil {
		return nil, err
	}

	i.SetBytes(b)
	return i, nil
}
