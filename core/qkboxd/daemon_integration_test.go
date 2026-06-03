package qkboxd

import (
	"context"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/shared/api"
)

func startDaemon(t *testing.T) *ipc.Client {
	t.Helper()
	t.Setenv("QKBOX_STATE_DIR", t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx)
	}()

	readyCtx, readyCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer readyCancel()
	if err := ipc.WaitForReady(readyCtx); err != nil {
		t.Fatalf("qkboxd did not become ready: %v", err)
	}
	return ipc.NewClient()
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

