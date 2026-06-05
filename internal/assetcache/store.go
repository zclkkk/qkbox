package assetcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Blob struct {
	Key       string
	SHA256    string
	SizeBytes int64
}

type Store struct {
	root string
}

func NewStore(stateDir string) *Store {
	return &Store{root: filepath.Join(stateDir, "assets")}
}

func (s *Store) Put(kind string, content []byte) (Blob, error) {
	if s == nil || s.root == "" {
		return Blob{}, fmt.Errorf("asset cache is not configured")
	}
	if !validKind(kind) {
		return Blob{}, fmt.Errorf("invalid asset kind %q", kind)
	}
	sum := sha256.Sum256(content)
	sha := hex.EncodeToString(sum[:])
	dir := filepath.Join(s.root, kind)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Blob{}, err
	}
	finalPath := filepath.Join(dir, sha)
	if _, err := os.Stat(finalPath); err == nil {
		return Blob{Key: filepath.ToSlash(filepath.Join(kind, sha)), SHA256: sha, SizeBytes: int64(len(content))}, nil
	} else if !os.IsNotExist(err) {
		return Blob{}, err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return Blob{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return Blob{}, err
	}
	if err := tmp.Close(); err != nil {
		return Blob{}, err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return Blob{}, err
	}
	return Blob{Key: filepath.ToSlash(filepath.Join(kind, sha)), SHA256: sha, SizeBytes: int64(len(content))}, nil
}

func validKind(kind string) bool {
	if kind == "" || strings.Contains(kind, "/") || strings.Contains(kind, `\`) || strings.Contains(kind, "..") {
		return false
	}
	for _, r := range kind {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
