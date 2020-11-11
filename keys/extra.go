package keys

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/xelaj/errs"
	"github.com/xelaj/go-dry"
)

func ReadFromFile(path string) ([]*rsa.PublicKey, error) {
	if !dry.FileExists(path) {
		return nil, errs.NotFound("file", path)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "reading file  keys")
	}
	keys := make([]*rsa.PublicKey, 0)
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}

		key, err := pemBytesToRsa(block.Bytes)
		if err != nil {
			const offset = 1 // +1 потому что считаем с 0
			return nil, errors.Wrapf(err, "decoding key №%d", len(keys)+offset)
		}

		keys = append(keys, key)
		data = rest
	}

	return keys, nil
}

func SaveRsaKey(key *rsa.PublicKey) string {
	data := x509.MarshalPKCS1PublicKey(key)
	buf := bytes.NewBufferString("")
	err := pem.Encode(buf, &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: data,
	})
	dry.PanicIfErr(err)

	return buf.String()
}
