package mtproto

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"

	"github.com/pkg/errors"
	"github.com/xelaj/go-dry"
	mtcrypto "github.com/xelaj/mtproto/crypto"
	ige "github.com/xelaj/mtproto/crypto/aes_ige"
	"github.com/xelaj/mtproto/encoding/tl"
	"github.com/xelaj/mtproto/keys"
	"github.com/xelaj/mtproto/service"
)

func handshake(conn net.Conn, publicKey *rsa.PublicKey) (*SessionCredentials, error) {
	nonceFirst := service.RandomInt128()
	pqParams := new(service.ResPQ)
	if err := sendUnencrypted(conn, &ReqPQParams{nonceFirst}, pqParams); err != nil {
		return nil, err
	}

	if nonceFirst.Cmp(pqParams.Nonce.Int) != 0 {
		return nil, errors.New("handshake: Wrong nonce")
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
		return nil, errors.New("handshake: Can't find fingerprint")
	}

	// (encoding) p_q_inner_data
	pq := big.NewInt(0).SetBytes(pqParams.Pq)
	p, q := splitPQ(pq)
	nonceSecond := service.RandomInt256()
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
	copy(hashAndMsg, append(dry.Sha1(string(message)), message...))

	fgpr, err := keys.RSAFingerprint(publicKey)
	if err != nil {
		return nil, err
	}

	var dhResponse service.ServerDHParams
	if err := sendUnencrypted(conn, &ReqDHParamsParams{
		Nonce:                nonceFirst,
		ServerNonce:          nonceServer,
		P:                    p.Bytes(),
		Q:                    q.Bytes(),
		PublicKeyFingerprint: int64(binary.LittleEndian.Uint64(fgpr)),
		EncryptedData:        doRSAencrypt(hashAndMsg, publicKey),
	}, &dhResponse); err != nil {
		return nil, err
	}

	dhParams, ok := dhResponse.(*service.ServerDHParamsOk)
	if !ok {
		return nil, fmt.Errorf("need *service.ServerDHParamsOk, got: %T", dhResponse)
	}

	if nonceFirst.Cmp(dhParams.Nonce.Int) != 0 {
		return nil, errors.New("handshake: Wrong nonce")
	}

	if nonceServer.Cmp(dhParams.ServerNonce.Int) != 0 {
		return nil, errors.New("handshake: Wrong server_nonce")
	}

	// проверку по хешу, удаление рандомных байт происходит в этой функции
	decodedMessage := ige.DecryptMessageWithTempKeys(dhParams.EncryptedAnswer, nonceSecond.Int, nonceServer.Int)
	dhi := new(service.ServerDHInnerData)
	if err := tl.Decode(decodedMessage, dhi); err != nil {
		return nil, err
	}

	if nonceFirst.Cmp(dhi.Nonce.Int) != 0 {
		return nil, errors.New("Handshake: Wrong nonce")
	}
	if nonceServer.Cmp(dhi.ServerNonce.Int) != 0 {
		return nil, errors.New("Handshake: Wrong server_nonce")
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
	copy(t4[33:], dry.Sha1Byte(authKey)[0:8])
	nonceHash1 := dry.Sha1Byte(t4)[4:20]
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
	if err := sendUnencrypted(conn, &SetClientDHParamsParams{
		Nonce:         nonceFirst,
		ServerNonce:   nonceServer,
		EncryptedData: ige.EncryptMessageWithTempKeys(clientDHDataMsg, nonceSecond.Int, nonceServer.Int),
	}, &dhGenStatus); err != nil {
		return nil, errors.Wrap(err, "sending clientDHParams")
	}

	dhg, ok := dhGenStatus.(*service.DHGenOk)
	if !ok {
		return nil, errors.New("Handshake: Need DHGenOk")
	}
	if nonceFirst.Cmp(dhg.Nonce.Int) != 0 {
		return nil, fmt.Errorf("Handshake: Wrong nonce: %v, %v", nonceFirst, dhg.Nonce)
	}
	if nonceServer.Cmp(dhg.ServerNonce.Int) != 0 {
		return nil, fmt.Errorf("Handshake: Wrong server_nonce: %v, %v", nonceServer, dhg.ServerNonce)
	}
	if !bytes.Equal(nonceHash1, dhg.NewNonceHash1.Bytes()) {
		return nil, fmt.Errorf(
			"handshake: Wrong new_nonce_hash1: %v, %v",
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
		return errors.Wrap(err, "sending data")
	}

	_, err = conn.Write(data)
	if err != nil {
		return errors.Wrap(err, "sending request")
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
