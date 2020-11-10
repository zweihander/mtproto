package mtproto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type SessionStore interface {
	Get() (*SessionCredentials, error)
	Set(*SessionCredentials) error
}

type SessionCredentials struct {
	AuthKey     []byte `json:"auth_key"`
	AuthKeyHash []byte `json:"auth_key_hash"`
	ServerSalt  int64  `json:"server_salt"`
}

var ErrNoCredentials = fmt.Errorf("no credentials")

type noOpSessionStore struct {
}

func (*noOpSessionStore) Get() (*SessionCredentials, error) {
	return nil, ErrNoCredentials
}

func (*noOpSessionStore) Set(*SessionCredentials) error { return nil }

type FileSessionStore struct {
	path string
}

func NewFileSessionStore(path string) *FileSessionStore {
	return &FileSessionStore{path}
}

func (fs *FileSessionStore) Get() (*SessionCredentials, error) {
	f, err := os.Open(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoCredentials
		}

		return nil, err
	}
	defer f.Close()

	creds := new(SessionCredentials)
	if err := json.NewDecoder(f).Decode(creds); err != nil {
		return nil, err
	}

	return creds, nil
}

func (fs *FileSessionStore) Set(sess *SessionCredentials) error {
	f, err := os.Open(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(fs.path), 0770); err != nil {
				return err
			}

			f, err = os.Create(fs.path)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(sess)
}
