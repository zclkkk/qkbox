//go:build windows

package provideripc

import (
	"context"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

func Listen(endpoint string) (net.Listener, error) {
	return winio.ListenPipe(endpoint, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;OW)",
	})
}

func Dial(ctx context.Context, endpoint string) (net.Conn, error) {
	return winio.DialPipeContext(ctx, endpoint)
}

func DefaultEndpoint() string {
	if endpoint := os.Getenv(EnvEndpoint); endpoint != "" {
		return endpoint
	}
	return `\\.\pipe\qkbox-provider`
}
