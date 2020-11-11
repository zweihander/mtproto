package mtcrypto

import (
	"crypto/rand"
	"crypto/sha1"
)

func AuthKeyHash(key []byte) []byte {
	r := sha1.Sum(key)
	return r[12:20]
}

func Sha1Bytes(input []byte) []byte {
	r := sha1.Sum(input)
	return r[:]
}

func RandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}
