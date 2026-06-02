package qkboxd

import (
	"context"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/shared/api"
)

func TestDaemonHelloOverLocalTransport(t *testing.T) {
	t.Setenv("QKBOX_STATE_DIR", t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx)
	}()

	readyCtx, readyCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer readyCancel()
	if err := ipc.WaitForReady(readyCtx); err != nil {
		cancel()
		t.Fatalf("qkboxd did not become ready: %v", err)
	}

	client := ipc.NewClient()
	reply, structured := client.Hello(context.Background(), api.DefaultHelloRequest())
	if structured != nil {
		t.Fatalf("hello returned structured error: %v", structured)
	}
	if reply.APIVersion != api.APIVersion {
		t.Fatalf("api version = %s", reply.APIVersion)
	}
	if len(reply.RuntimeCapabilities) == 0 || len(reply.PlatformCapabilities) == 0 {
		t.Fatal("expected capability shells")
	}

	bad := api.DefaultHelloRequest()
	bad.ClientAPIVersion = "0"
	_, structured = client.Hello(context.Background(), bad)
	if structured == nil {
		t.Fatal("expected version mismatch error")
	}
	if structured.Code != api.ErrorIPCVersionUnsupported {
		t.Fatalf("error code = %s", structured.Code)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("daemon exited with error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not stop")
	}
}
