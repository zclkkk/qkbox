package qkboxd

import (
	"fmt"
	"os"
	"path/filepath"
)

type UserLock struct {
	file *os.File
	path string
}

func AcquireUserLock() (*UserLock, error) {
	dir, err := userStateDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "qkboxd.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("qkboxd already appears to be running for this user: %w", err)
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	_ = file.Sync()
	return &UserLock{file: file, path: path}, nil
}

func (l *UserLock) Release() {
	if l == nil {
		return
	}
	if l.file != nil {
		_ = l.file.Close()
	}
	if l.path != "" {
		_ = os.Remove(l.path)
	}
}

func userStateDir() (string, error) {
	if dir := os.Getenv("QKBOX_STATE_DIR"); dir != "" {
		return dir, os.MkdirAll(dir, 0o700)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "qkbox")
	return dir, os.MkdirAll(dir, 0o700)
}
