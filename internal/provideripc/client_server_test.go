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

func TestProviderMethodRegistryIncludesRuntimeMethods(t *testing.T) {
	for _, method := range []string{
		MethodAuth,
		MethodGetStatus,
		MethodPrepareFeature,
		MethodRunRepairAction,
		MethodRuntimeStart,
		MethodRuntimeStop,
		MethodRuntimeHeartbeat,
		MethodRuntimeGetStatus,
		MethodRuntimeGetRuntimeCapabilities,
		MethodRuntimeGetTraffic,
		MethodRuntimeGetConnections,
		MethodRuntimeListGroups,
		MethodRuntimeSelectOutbound,
		MethodRuntimeURLTest,
		MethodRuntimeCloseConnection,
		MethodRuntimeCloseAllConnections,
		MethodRuntimeListenerInfo,
		MethodRuntimeSubscribeEvents,
	} {
		if _, ok := MethodRegistry[method]; !ok {
			t.Fatalf("missing provider method %s", method)
		}
	}
}

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

func (testHandler) RuntimeStart(_ context.Context, req RuntimeStartRequest) (RuntimeStartReply, *api.StructuredError) {
	return RuntimeStartReply{OwnerState: api.ProviderOwnerState{
		Owned:      true,
		SessionID:  req.SessionID,
		RuntimeID:  req.RuntimeID,
		SnapshotID: req.SnapshotID,
		Mode:       req.Mode,
	}}, nil
}

func (testHandler) RuntimeStop(context.Context, RuntimeStopRequest) (RuntimeStopReply, *api.StructuredError) {
	return RuntimeStopReply{}, nil
}

func (testHandler) RuntimeHeartbeat(_ context.Context, req RuntimeHeartbeatRequest) (RuntimeHeartbeatReply, *api.StructuredError) {
	return RuntimeHeartbeatReply{OwnerState: api.ProviderOwnerState{Owned: true, SessionID: req.SessionID, RuntimeID: req.RuntimeID}}, nil
}

func (testHandler) RuntimeGetStatus(context.Context, RuntimeGetStatusRequest) (RuntimeGetStatusReply, *api.StructuredError) {
	return RuntimeGetStatusReply{}, nil
}

func (testHandler) RuntimeGetRuntimeCapabilities(context.Context, RuntimeGetRuntimeCapabilitiesRequest) (RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return RuntimeGetRuntimeCapabilitiesReply{Capabilities: api.RuntimeCapabilityShell()}, nil
}

func (testHandler) RuntimeGetTraffic(context.Context, RuntimeGetTrafficRequest) (RuntimeGetTrafficReply, *api.StructuredError) {
	return RuntimeGetTrafficReply{}, nil
}

func (testHandler) RuntimeGetConnections(context.Context, RuntimeGetConnectionsRequest) (RuntimeGetConnectionsReply, *api.StructuredError) {
	return RuntimeGetConnectionsReply{}, nil
}

func (testHandler) RuntimeListGroups(context.Context, RuntimeListGroupsRequest) (RuntimeListGroupsReply, *api.StructuredError) {
	return RuntimeListGroupsReply{}, nil
}

func (testHandler) RuntimeSelectOutbound(context.Context, RuntimeSelectOutboundRequest) (RuntimeSelectOutboundReply, *api.StructuredError) {
	return RuntimeSelectOutboundReply{}, nil
}

func (testHandler) RuntimeURLTest(context.Context, RuntimeURLTestRequest) (RuntimeURLTestReply, *api.StructuredError) {
	return RuntimeURLTestReply{}, nil
}

func (testHandler) RuntimeCloseConnection(context.Context, RuntimeCloseConnectionRequest) (RuntimeCloseConnectionReply, *api.StructuredError) {
	return RuntimeCloseConnectionReply{}, nil
}

func (testHandler) RuntimeCloseAllConnections(context.Context, RuntimeCloseAllConnectionsRequest) (RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	return RuntimeCloseAllConnectionsReply{}, nil
}

func (testHandler) RuntimeListenerInfo(context.Context, RuntimeListenerInfoRequest) (RuntimeListenerInfoReply, *api.StructuredError) {
	return RuntimeListenerInfoReply{}, nil
}

func (testHandler) RuntimeSubscribeEvents(ctx context.Context, _ RuntimeSubscribeEventsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	ch := make(chan api.RuntimeEvent, 1)
	ch <- api.RuntimeEvent{Event: api.EventEngineLog, Data: api.RuntimeLogEntry{Source: "test", Level: "info", Message: "hello"}}
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
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

func TestProviderIPCEventSubscriptionRoundTrip(t *testing.T) {
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
	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	events, structured := client.RuntimeSubscribeEvents(subCtx, RuntimeSubscribeEventsRequest{SessionID: "session", RuntimeID: "runtime"})
	if structured != nil {
		t.Fatalf("subscribe: %v", structured)
	}

	select {
	case event := <-events:
		if event.Event != api.EventEngineLog {
			t.Fatalf("event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
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
