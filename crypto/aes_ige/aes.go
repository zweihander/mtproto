package ige

import (
	"bytes"
	"crypto/aes"
	"crypto/sha1"
	"math/big"

	mtcrypto "github.com/xelaj/mtproto/crypto"
)

type AesBlock [aes.BlockSize]byte
type AesKV [32]byte
type AesIgeBlock [48]byte

func MessageKey(msg []byte) []byte {
	r := sha1.Sum(msg)
	return r[4:20]
}

func Encrypt(msg, key []byte) ([]byte, error) {
	msgKey := MessageKey(msg)
	aesKey, aesIV := generateAESIGE(msgKey, key, false)

	// СУДЯ ПО ВСЕМУ вообще не уверен, но это видимо паддинг для добива блока, чтоб он делился на 256 бит
	data := make([]byte, len(msg)+((16-(len(msg)%16))&15))
	copy(data, msg)

	c, err := NewCipher(aesKey, aesIV)
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(data))
	if err := c.doAES256IGEencrypt(data, out); err != nil {
		return nil, err
	}

	return out, nil
}

// checkData это msgkey в понятиях мтпрото, нужно что бы проверить, успешно ли прошла расшифровка
func Decrypt(msg, key, checkData []byte) ([]byte, error) {
	aesKey, aesIV := generateAESIGE(checkData, key, true)

	c, err := NewCipher(aesKey, aesIV)
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(msg))
	if err := c.doAES256IGEdecrypt(msg, out); err != nil {
		return nil, err
	}

	return out, nil
}

func doAES256IGEencrypt(data, out, key, iv []byte) error {
	c, err := NewCipher(key, iv)
	if err != nil {
		return err
	}
	return c.doAES256IGEencrypt(data, out)
}

func doAES256IGEdecrypt(data, out, key, iv []byte) error {
	c, err := NewCipher(key, iv)
	if err != nil {
		return err
	}
	return c.doAES256IGEdecrypt(data, out)
}

// DecryptMessageWithTempKeys дешифрует сообщение паролем, которые получены в процессе обмена ключами диффи хеллмана
func DecryptMessageWithTempKeys(msg []byte, nonceSecond, nonceServer *big.Int) ([]byte, error) {
	key, iv := generateTempKeys(nonceSecond, nonceServer)
	decodedWithHash := make([]byte, len(msg))
	if err := doAES256IGEdecrypt(msg, decodedWithHash, key, iv); err != nil {
		return nil, err
	}

	// decodedWithHash := SHA1(answer) + answer + (0-15 рандомных байт); длина должна делиться на 16;
	decodedHash := decodedWithHash[:20]
	decodedMessage := decodedWithHash[20:]

	// режем последние 0-15 байт ориентируюясь по хешу
	for i := len(decodedMessage) - 1; i > len(decodedMessage)-16; i-- {
		r := sha1.Sum(decodedMessage[:i])
		if bytes.Equal(decodedHash, r[:]) {
			return decodedMessage[:i], nil
		}
	}

	panic("couldn't trim message: hashes incompatible on more than 16 tries")
}

// EncryptMessageWithTempKeys шифрует сообщение паролем, которые получены в процессе обмена ключами диффи хеллмана
func EncryptMessageWithTempKeys(msg []byte, nonceSecond, nonceServer *big.Int) ([]byte, error) {
	hash := sha1.Sum(msg)

	// добавляем остаток рандомных байт в сообщение, что бы суммарно оно делилось на 16
	totalLen := len(hash) + len(msg)
	overflowedLen := totalLen % 16
	needToAdd := 16 - overflowedLen

	randb, err := mtcrypto.RandomBytes(needToAdd)
	if err != nil {
		return nil, err
	}

	msg = bytes.Join([][]byte{hash[:], msg, randb}, []byte{})
	key, iv := generateTempKeys(nonceSecond, nonceServer)
	encodedWithHash := make([]byte, len(msg))
	if err := doAES256IGEencrypt(msg, encodedWithHash, key, iv); err != nil {
		return nil, err
	}

	return encodedWithHash, nil
}

// https://tlgrm.ru/docs/mtproto/auth_key#server-otvecaet-dvuma-sposobami
// generateTempKeys генерирует временные ключи для шифрования в процессе обемна ключами.
func generateTempKeys(nonceSecond, nonceServer *big.Int) (key, iv []byte) {
	// nonceSecond + nonceServer
	t1 := make([]byte, 48)
	copy(t1[0:], nonceSecond.Bytes())
	copy(t1[32:], nonceServer.Bytes())
	// SHA1(nonceSecond + nonceServer)
	hash1 := sha1.Sum(t1)

	// nonceServer + nonceSecond
	t2 := make([]byte, 48)
	copy(t2[0:], nonceServer.Bytes())
	copy(t2[16:], nonceSecond.Bytes())
	// SHA1(nonceServer + nonceSecond)
	hash2 := sha1.Sum(t2)

	// SHA1(nonceSecond + nonceServer) + substr (SHA1(nonceServer + nonceSecond), 0, 12);
	tmpAESKey := make([]byte, 32)
	// SHA1(nonceSecond + nonceServer)
	copy(tmpAESKey[0:], hash1[:])
	// substr (SHA1(nonceServer + nonceSecond), 0, 12)
	copy(tmpAESKey[20:], hash2[0:12])

	t3 := make([]byte, 64) // nonceSecond + nonceSecond
	copy(t3[0:], nonceSecond.Bytes())
	copy(t3[32:], nonceSecond.Bytes())
	hash3 := sha1.Sum(t3) // SHA1(nonceSecond + nonceSecond)

	// substr (SHA1(server_nonce + new_nonce), 12, 8) + SHA1(new_nonce + new_nonce) + substr (new_nonce, 0, 4);
	tmpAESIV := make([]byte, 32)
	// substr (SHA1(nonceServer + nonceSecond), 12, 8)
	copy(tmpAESIV[0:], hash2[12:12+8])
	// SHA1(nonceSecond + nonceSecond)
	copy(tmpAESIV[8:], hash3[:])
	// substr (nonceSecond, 0, 4)
	copy(tmpAESIV[28:], nonceSecond.Bytes()[0:4])

	return tmpAESKey, tmpAESIV
}
