package keys

import (
	"bytes"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"fmt"
	"math/big"

	"github.com/xelaj/mtproto/encoding/tl"
)

// RSAFingerprint вычисляет отпечаток ключа
// т.к. rsa ключ в понятиях MTProto это TL объект, то используется буффер
// подробнее https://core.telegram.org/mtproto/auth_key
func RSAFingerprint(key *rsa.PublicKey) ([]byte, error) {
	if key == nil {
		return nil, fmt.Errorf("key can't be nil")
	}

	exponentAsBigInt := (big.NewInt(0)).SetInt64(int64(key.E))

	buf := bytes.NewBuffer(nil)
	w := tl.NewWriteCursor(buf)
	if err := w.PutMessage(key.N.Bytes()); err != nil {
		return nil, err
	}
	if err := w.PutMessage(exponentAsBigInt.Bytes()); err != nil {
		return nil, err
	}

	sh := sha1.Sum(buf.Bytes())
	return sh[12:], nil // последние 8 байт это и есть отпечаток
}

func pemBytesToRsa(data []byte) (*rsa.PublicKey, error) {
	key, err := x509.ParsePKCS1PublicKey(data)
	if err == nil {
		return key, nil
	}

	if err.Error() == "x509: failed to parse public key (use ParsePKIXPublicKey instead for this key format)" {
		var k interface{}
		k, err = x509.ParsePKIXPublicKey(data)
		if err == nil {
			return k.(*rsa.PublicKey), nil
		}
	}

	return nil, err
}
