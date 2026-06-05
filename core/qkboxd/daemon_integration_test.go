package qkboxd

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/shared/api"
)

func startDaemon(t *testing.T) *ipc.Client {
	t.Helper()
	return startDaemonWithStateDir(t, t.TempDir())
}

func startDaemonWithStateDir(t *testing.T, stateDir string) *ipc.Client {
	t.Helper()
	t.Setenv("QKBOX_STATE_DIR", stateDir)
	t.Setenv("QKBOX_IPC_ENDPOINT_ID", fmt.Sprintf("test-%d", time.Now().UnixNano()))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("qkboxd exit: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("qkboxd did not stop")
		}
	})

	readyCtx, readyCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer readyCancel()
	if err := ipc.WaitForReady(readyCtx); err != nil {
		t.Fatalf("qkboxd did not become ready: %v", err)
	}
	return ipc.NewClient()
}

type integrationProviderHandler struct{}

func (integrationProviderHandler) GetStatus(context.Context, struct{}) (provideripc.StatusReply, *api.StructuredError) {
	return provideripc.StatusReply{Version: api.QKBoxDVersion}, nil
}

func (integrationProviderHandler) PrepareFeature(_ context.Context, req api.PrepareFeatureRequest) (api.PrepareFeatureReply, *api.StructuredError) {
	if req.Feature == api.CapabilityBackgroundService {
		return api.PrepareFeatureReply{Feature: req.Feature, State: api.CapabilityAvailable}, nil
	}
	return api.PrepareFeatureReply{Feature: req.Feature, State: api.CapabilityUnavailable, Reason: "not implemented"}, nil
}

func (integrationProviderHandler) RunRepairAction(_ context.Context, req api.RunRepairActionRequest) (api.RunRepairActionReply, *api.StructuredError) {
	return api.RunRepairActionReply{Action: req.Action, Outcome: "success"}, nil
}

func (integrationProviderHandler) RuntimeStart(_ context.Context, req provideripc.RuntimeStartRequest) (provideripc.RuntimeStartReply, *api.StructuredError) {
	return provideripc.RuntimeStartReply{OwnerState: api.ProviderOwnerState{Owned: true, SessionID: req.SessionID, RuntimeID: req.RuntimeID, SnapshotID: req.SnapshotID, Mode: req.Mode}}, nil
}

func (integrationProviderHandler) RuntimeStop(context.Context, provideripc.RuntimeStopRequest) (provideripc.RuntimeStopReply, *api.StructuredError) {
	return provideripc.RuntimeStopReply{}, nil
}

func (integrationProviderHandler) RuntimeHeartbeat(_ context.Context, req provideripc.RuntimeHeartbeatRequest) (provideripc.RuntimeHeartbeatReply, *api.StructuredError) {
	return provideripc.RuntimeHeartbeatReply{OwnerState: api.ProviderOwnerState{Owned: true, SessionID: req.SessionID, RuntimeID: req.RuntimeID}}, nil
}

func (integrationProviderHandler) RuntimeGetStatus(context.Context, provideripc.RuntimeGetStatusRequest) (provideripc.RuntimeGetStatusReply, *api.StructuredError) {
	return provideripc.RuntimeGetStatusReply{}, nil
}

func (integrationProviderHandler) RuntimeGetRuntimeCapabilities(context.Context, provideripc.RuntimeGetRuntimeCapabilitiesRequest) (provideripc.RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return provideripc.RuntimeGetRuntimeCapabilitiesReply{Capabilities: api.RuntimeCapabilityShell()}, nil
}

func (integrationProviderHandler) RuntimeGetTraffic(context.Context, provideripc.RuntimeGetTrafficRequest) (provideripc.RuntimeGetTrafficReply, *api.StructuredError) {
	return provideripc.RuntimeGetTrafficReply{}, nil
}

func (integrationProviderHandler) RuntimeGetConnections(context.Context, provideripc.RuntimeGetConnectionsRequest) (provideripc.RuntimeGetConnectionsReply, *api.StructuredError) {
	return provideripc.RuntimeGetConnectionsReply{}, nil
}

func (integrationProviderHandler) RuntimeListGroups(context.Context, provideripc.RuntimeListGroupsRequest) (provideripc.RuntimeListGroupsReply, *api.StructuredError) {
	return provideripc.RuntimeListGroupsReply{}, nil
}

func (integrationProviderHandler) RuntimeSelectOutbound(context.Context, provideripc.RuntimeSelectOutboundRequest) (provideripc.RuntimeSelectOutboundReply, *api.StructuredError) {
	return provideripc.RuntimeSelectOutboundReply{}, nil
}

func (integrationProviderHandler) RuntimeURLTest(context.Context, provideripc.RuntimeURLTestRequest) (provideripc.RuntimeURLTestReply, *api.StructuredError) {
	return provideripc.RuntimeURLTestReply{}, nil
}

func (integrationProviderHandler) RuntimeCloseConnection(context.Context, provideripc.RuntimeCloseConnectionRequest) (provideripc.RuntimeCloseConnectionReply, *api.StructuredError) {
	return provideripc.RuntimeCloseConnectionReply{}, nil
}

func (integrationProviderHandler) RuntimeCloseAllConnections(context.Context, provideripc.RuntimeCloseAllConnectionsRequest) (provideripc.RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	return provideripc.RuntimeCloseAllConnectionsReply{}, nil
}

func (integrationProviderHandler) RuntimeListenerInfo(context.Context, provideripc.RuntimeListenerInfoRequest) (provideripc.RuntimeListenerInfoReply, *api.StructuredError) {
	return provideripc.RuntimeListenerInfoReply{}, nil
}

func (integrationProviderHandler) RuntimeSubscribeEvents(ctx context.Context, _ provideripc.RuntimeSubscribeEventsRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	ch := make(chan api.RuntimeEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func startProviderForDaemonTest(t *testing.T, stateDir string) {
	t.Helper()
	endpoint := daemonTestProviderEndpoint(t)
	clientPath := provideripc.ClientConfigPath(stateDir)
	serverPath := provideripc.ServerConfigPath(stateDir)
	if _, _, err := provideripc.WriteConfigPair(clientPath, serverPath, endpoint); err != nil {
		t.Fatal(err)
	}
	cfg, err := provideripc.ReadServerConfig(serverPath)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := provideripc.Listen(cfg.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- provideripc.NewServer(cfg.Token, integrationProviderHandler{}).Serve(ctx, listener)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("provider exit: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("provider did not stop")
		}
	})
}

func daemonTestProviderEndpoint(t *testing.T) string {
	t.Helper()
	id := fmt.Sprintf("qkbox-provider-daemon-test-%d", time.Now().UnixNano())
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + id
	}
	return filepath.Join(t.TempDir(), id+".sock")
}

func TestDaemonHelloOverLocalTransport(t *testing.T) {
	client := startDaemon(t)

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
}

func TestDaemonPrivilegedProviderStatusOverLocalTransport(t *testing.T) {
	stateDir := t.TempDir()
	startProviderForDaemonTest(t, stateDir)
	client := startDaemonWithStateDir(t, stateDir)

	status, structured := client.PlatformGetPrivilegedProviderStatus(context.Background(), api.GetPrivilegedProviderStatusRequest{})
	if structured != nil {
		t.Fatalf("provider status: %v", structured)
	}
	if !status.Status.Installed || !status.Status.Reachable || !status.Status.Authenticated {
		t.Fatalf("status = %+v", status.Status)
	}

	prepared, structured := client.PlatformPrepareFeature(context.Background(), api.PrepareFeatureRequest{Feature: api.CapabilityBackgroundService})
	if structured != nil {
		t.Fatalf("prepare: %v", structured)
	}
	if prepared.State != api.CapabilityAvailable {
		t.Fatalf("prepared = %+v", prepared)
	}
}

func TestDaemonProfileFlow(t *testing.T) {
	client := startDaemon(t)
	ctx := context.Background()

	// create profile
	createReply, structured := client.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "integration-test",
		Content: `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`,
	})
	if structured != nil {
		t.Fatalf("create: %v", structured)
	}
	pid := createReply.Profile.ID

	// get profile with content
	getReply, structured := client.GetProfile(ctx, api.GetProfileRequest{ProfileID: pid})
	if structured != nil {
		t.Fatalf("get: %v", structured)
	}
	if getReply.Content == "" {
		t.Fatal("expected content in get reply")
	}

	// list
	listReply, structured := client.ListProfiles(ctx, api.ListProfilesRequest{})
	if structured != nil {
		t.Fatalf("list: %v", structured)
	}
	if len(listReply.Profiles) != 1 {
		t.Fatalf("count = %d", len(listReply.Profiles))
	}

	// validate
	validReply, structured := client.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if structured != nil {
		t.Fatalf("validate: %v", structured)
	}
	if validReply.Diagnostics.Status != "valid" {
		t.Fatalf("status = %s", validReply.Diagnostics.Status)
	}

	// create snapshot
	snapReply, structured := client.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if structured != nil {
		t.Fatalf("snapshot: %v", structured)
	}
	sid := snapReply.Snapshot.ID

	// activate
	_, structured = client.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: sid})
	if structured != nil {
		t.Fatalf("activate: %v", structured)
	}

	// get active profile
	activeReply, structured := client.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if structured != nil {
		t.Fatalf("get active: %v", structured)
	}
	if activeReply.Profile == nil || activeReply.Profile.ID != pid {
		t.Fatal("wrong active profile")
	}

	// get active snapshot
	activeSnapReply, structured := client.GetActiveSnapshot(ctx, api.GetActiveSnapshotRequest{})
	if structured != nil {
		t.Fatalf("get active snap: %v", structured)
	}
	if activeSnapReply.Snapshot == nil || activeSnapReply.Snapshot.ID != sid {
		t.Fatal("wrong active snapshot")
	}

	// rollback
	_, structured = client.RollbackToSnapshot(ctx, api.RollbackToSnapshotRequest{SnapshotID: sid})
	if structured != nil {
		t.Fatalf("rollback: %v", structured)
	}

	// delete blocked by active snapshot
	_, structured = client.DeleteProfile(ctx, api.DeleteProfileRequest{ProfileID: pid})
	if structured == nil {
		t.Fatal("expected error deleting profile with active snapshot")
	}
	if structured.Code != api.ErrorProfileHasSnapshot {
		t.Fatalf("code = %s", structured.Code)
	}
}

func TestDaemonValidationBlocksInvalidSnapshot(t *testing.T) {
	client := startDaemon(t)
	ctx := context.Background()

	createReply, structured := client.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "bad",
		Content: `not json`,
	})
	if structured != nil {
		t.Fatalf("create: %v", structured)
	}

	_, structured = client.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{
		ProfileID: createReply.Profile.ID,
	})
	if structured == nil {
		t.Fatal("expected validation error")
	}
	if structured.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", structured.Code)
	}
}

func TestDaemonEngineLifecycle(t *testing.T) {
	client := startDaemon(t)
	ctx := context.Background()

	// 1. Create profile with minimal direct config
	createReply, structured := client.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "engine-test",
		Content: `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`,
	})
	if structured != nil {
		t.Fatalf("create: %v", structured)
	}
	pid := createReply.Profile.ID

	// 2. Create snapshot
	snapReply, structured := client.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if structured != nil {
		t.Fatalf("snapshot: %v", structured)
	}
	sid := snapReply.Snapshot.ID

	// 3. Activate snapshot
	_, structured = client.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: sid})
	if structured != nil {
		t.Fatalf("activate: %v", structured)
	}

	// 4. Start Engine
	_, structured = client.EngineStart(ctx, api.EngineStartRequest{})
	if structured != nil {
		t.Fatalf("engine start: %v", structured)
	}

	// 5. Get Status
	statusReply, structured := client.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if structured != nil {
		t.Fatalf("engine get status: %v", structured)
	}
	if statusReply.Status.State != "STARTED" {
		t.Fatalf("expected STARTED, got %s", statusReply.Status.State)
	}

	// 6. Stop Engine
	_, structured = client.EngineStop(ctx, api.EngineStopRequest{})
	if structured != nil {
		t.Fatalf("engine stop: %v", structured)
	}

	// 7. Get Status
	statusReply, structured = client.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if structured != nil {
		t.Fatalf("engine get status: %v", structured)
	}
	if statusReply.Status.State != "IDLE" {
		t.Fatalf("expected IDLE, got %s", statusReply.Status.State)
	}
}
