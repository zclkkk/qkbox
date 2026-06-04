//go:build !windows

package ipc

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

func Listen() (net.Listener, error) {
	path, err := socketPath()
	if err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if chmodErr := os.Chmod(path, 0o600); chmodErr != nil {
		_ = listener.Close()
		return nil, chmodErr
	}
	return listener, nil
}

func Dial(ctx context.Context) (net.Conn, error) {
	path, err := socketPath()
	if err != nil {
		return nil, err
	}
	var d net.Dialer
	return d.DialContext(ctx, "unix", path)
}

func Endpoint() string {
	path, err := socketPath()
	if err != nil {
		return ""
	}
	return path
}

func socketPath() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		base = cacheDir
	}
	dir := filepath.Join(base, "qkbox")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := "qkboxd.sock"
	if endpointID := os.Getenv("QKBOX_IPC_ENDPOINT_ID"); endpointID != "" {
		endpointID = regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(endpointID, "_")
		name = "qkboxd-" + endpointID + ".sock"
	}
	return filepath.Join(dir, name), nil
}

func WaitForReady(ctx context.Context) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		conn, err := Dial(ctx)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
