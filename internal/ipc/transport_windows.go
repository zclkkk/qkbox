//go:build windows

package ipc

import (
	"context"
	"net"
	"os"
	"os/user"
	"regexp"
	"time"

	"github.com/Microsoft/go-winio"
)

func Listen() (net.Listener, error) {
	return winio.ListenPipe(pipeName(), &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;OW)",
	})
}

func Dial(ctx context.Context) (net.Conn, error) {
	return winio.DialPipeContext(ctx, pipeName())
}

func Endpoint() string {
	return pipeName()
}

func pipeName() string {
	identity := "unknown"
	if endpointID := os.Getenv("QKBOX_IPC_ENDPOINT_ID"); endpointID != "" {
		identity = endpointID
	} else {
		current, err := user.Current()
		if err == nil && current.Uid != "" {
			identity = current.Uid
		}
	}
	identity = regexp.MustCompile(`[^A-Za-z0-9]+`).ReplaceAllString(identity, "_")
	return `\\.\pipe\qkbox-` + identity + `-qkboxd`
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
