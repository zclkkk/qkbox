//go:build !windows

package provideripc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func Listen(endpoint string) (net.Listener, error) {
	if info, err := os.Lstat(endpoint); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("provider endpoint exists and is not a socket: %s", endpoint)
		}
		if err := os.Remove(endpoint); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o700); err != nil {
		return nil, err
	}
	listener, err := net.Listen("unix", endpoint)
	if err != nil {
		return nil, err
	}
	if chmodErr := os.Chmod(endpoint, 0o600); chmodErr != nil {
		_ = listener.Close()
		return nil, chmodErr
	}
	return listener, nil
}

func Dial(ctx context.Context, endpoint string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "unix", endpoint)
}

func DefaultEndpoint() string {
	if endpoint := os.Getenv(EnvEndpoint); endpoint != "" {
		return endpoint
	}
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "qkbox", "qkbox-provider.sock")
}
