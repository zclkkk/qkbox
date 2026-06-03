package crypto

import (
	"fmt"
	"os"
	"path/filepath"
)

type SecretStore interface {
	GetOrCreateKey() ([]byte, error)
}

type FileKeyStore struct {
	path string
}

func NewFileKeyStore(stateDir string) *FileKeyStore {
	return &FileKeyStore{path: filepath.Join(stateDir, "master.key")}
}

func (s *FileKeyStore) GetOrCreateKey() ([]byte, error) {
	key, err := os.ReadFile(s.path)
	if err == nil {
		if len(key) != KeySize {
			return nil, fmt.Errorf("invalid key size: %d", len(key))
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	key, err = RandomBytes(KeySize)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(s.path, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}
