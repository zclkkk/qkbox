package provideripc

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

type testHandler struct{}

func (testHandler) GetStatus(context.Context, struct{}) (StatusReply, *api.StructuredError) {
	return StatusReply{Version: "test-version"}, nil
}

func (testHandler) PrepareFeature(_ context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	return api.PrepareFeatureReply{Feature: req.Feature, State: api.CapabilityAvailable}, nil
}

func (testHandler) RunRepairAction(_ context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	if req.Action != "test_repair_action" {
		return api.RunRepairActionReply{}, api.NewStructuredError(api.ErrorPlatformRepairActionNotFound, "missing", "provider", true)
	}
	return api.RunRepairActionReply{Action: req.Action, Outcome: "success"}, nil
}

func TestProviderIPCRoundTrip(t *testing.T) {
	endpoint := providerTestEndpoint(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- NewServer("token", testHandler{}).Serve(ctx, listener)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("server exit: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("provider server did not stop")
		}
	})

	client := NewClient(&ClientConfig{Endpoint: endpoint, Token: "token", ExpectedVersion: "test-version"})
	status, structured := client.GetStatus(context.Background())
	if structured != nil {
		t.Fatalf("status: %v", structured)
	}
	if status.Version != "test-version" {
		t.Fatalf("version = %s", status.Version)
	}

	prepared, structured := client.PrepareFeature(context.Background(), api.PrepareFeatureRequest{Feature: api.CapabilityBackgroundService})
	if structured != nil {
		t.Fatalf("prepare: %v", structured)
	}
	if prepared.State != api.CapabilityAvailable {
		t.Fatalf("prepared = %+v", prepared)
	}
}

func TestProviderIPCRejectsBadToken(t *testing.T) {
	endpoint := providerTestEndpoint(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- NewServer("token", testHandler{}).Serve(ctx, listener)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("server exit: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("provider server did not stop")
		}
	})

	client := NewClient(&ClientConfig{Endpoint: endpoint, Token: "bad-token", ExpectedVersion: "test-version"})
	_, structured := client.GetStatus(context.Background())
	if structured == nil || structured.Code != api.ErrorPlatformProviderAuthFailed {
		t.Fatalf("expected auth failed, got %v", structured)
	}
}

func TestProviderIPCServerClosesIdleConnectionOnShutdown(t *testing.T) {
	endpoint := providerTestEndpoint(t)
	ctx, cancel := context.WithCancel(context.Background())
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- NewServer("token", testHandler{}).Serve(ctx, listener)
	}()

	conn, err := Dial(context.Background(), endpoint)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	defer conn.Close()

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server exit: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("provider server did not stop with an idle accepted connection")
	}
}

func TestProviderIPCServerAuthReadDeadline(t *testing.T) {
	endpoint := providerTestEndpoint(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listener, err := Listen(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer("token", testHandler{})
	server.ioTimeout = 50 * time.Millisecond
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx, listener)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("server exit: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("provider server did not stop")
		}
	})

	conn, err := Dial(context.Background(), endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))

	var resp Response
	if err := ReadFrame(conn, &resp); err != nil {
		t.Fatalf("read deadline response: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != api.ErrorIPCInvalidRequest {
		t.Fatalf("deadline response = %+v", resp)
	}
}

func providerTestEndpoint(t *testing.T) string {
	t.Helper()
	id := fmt.Sprintf("qkbox-provider-test-%d", time.Now().UnixNano())
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + id
	}
	return filepath.Join(t.TempDir(), id+".sock")
}
