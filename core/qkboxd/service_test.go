package qkboxd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	qkboxcrypto "github.com/zclkkk/qkbox/internal/crypto"
	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/platform/capability"
	"github.com/zclkkk/qkbox/shared/api"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithPlatform(t, nil, nil)
}

func newTestServiceWithProxy(t *testing.T, proxy capability.SystemProxyProvider) *Service {
	t.Helper()
	return newTestServiceWithPlatform(t, proxy, nil)
}

func newTestServiceWithPlatform(t *testing.T, proxy capability.SystemProxyProvider, privileged capability.PrivilegedProvider) *Service {
	t.Helper()
	dir := t.TempDir()
	db, err := persistence.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	key, err := qkboxcrypto.RandomBytes(qkboxcrypto.KeySize)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(context.Background(), db, key, proxy, privileged)
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}

type fakeSystemProxy struct {
	availability  capability.SystemProxyAvailability
	state         capability.SystemProxyCurrentState
	currentErr    error
	snapshot      *capability.SystemProxySnapshot
	applyErr      error
	restoreErr    error
	snapshotCalls int
	applyCalls    int
	restoreCalls  int
	appliedAddr   string
	appliedPort   int
}

func (f *fakeSystemProxy) Availability() capability.SystemProxyAvailability {
	if !f.availability.Available && !f.availability.Supported && f.availability.Reason == "" {
		return capability.SystemProxyAvailability{Available: true, Supported: true}
	}
	return f.availability
}

func (f *fakeSystemProxy) Snapshot() (*capability.SystemProxySnapshot, error) {
	f.snapshotCalls++
	if f.snapshot != nil {
		return f.snapshot, nil
	}
	raw, _ := json.Marshal(f.state)
	return &capability.SystemProxySnapshot{Raw: raw}, nil
}

func (f *fakeSystemProxy) Apply(addr string, port int) error {
	f.applyCalls++
	f.appliedAddr = addr
	f.appliedPort = port
	if f.applyErr != nil {
		return f.applyErr
	}
	f.state = capability.SystemProxyCurrentState{Enabled: true, Addr: addr, Port: port}
	return nil
}

func (f *fakeSystemProxy) Restore(snapshot *capability.SystemProxySnapshot) error {
	f.restoreCalls++
	if f.restoreErr != nil {
		return f.restoreErr
	}
	f.state = capability.SystemProxyCurrentState{}
	return nil
}

func (f *fakeSystemProxy) CurrentState() (capability.SystemProxyCurrentState, error) {
	if f.currentErr != nil {
		return capability.SystemProxyCurrentState{}, f.currentErr
	}
	return f.state, nil
}

type fakePrivilegedProvider struct {
	status              api.PrivilegedProviderStatus
	statusWaitForCancel bool
	prepare             map[string]api.PrepareFeatureReply
	prepareErr          *api.StructuredError
	prepared            []string
	repairErr           *api.StructuredError
	repairCalls         []string
}

func readyFakePrivilegedProvider() *fakePrivilegedProvider {
	return &fakePrivilegedProvider{
		status: api.PrivilegedProviderStatus{
			Installed:       true,
			Reachable:       true,
			Authenticated:   true,
			Version:         api.QKBoxDVersion,
			ExpectedVersion: api.QKBoxDVersion,
			Endpoint:        "test-provider",
		},
	}
}

func (f *fakePrivilegedProvider) Status(ctx context.Context) api.PrivilegedProviderStatus {
	if f.statusWaitForCancel {
		<-ctx.Done()
		status := api.PrivilegedProviderStatus{
			Installed:       f.status.Installed,
			Endpoint:        f.status.Endpoint,
			ExpectedVersion: f.status.ExpectedVersion,
		}
		status.Reason = ctx.Err().Error()
		return status
	}
	return f.status
}

func (f *fakePrivilegedProvider) PrepareFeature(_ context.Context, feature string) (api.PrepareFeatureReply, *api.StructuredError) {
	f.prepared = append(f.prepared, feature)
	if f.prepareErr != nil {
		return api.PrepareFeatureReply{}, f.prepareErr
	}
	if f.prepare != nil {
		if reply, ok := f.prepare[feature]; ok {
			return reply, nil
		}
	}
	switch feature {
	case api.CapabilityBackgroundService:
		return api.PrepareFeatureReply{Feature: feature, State: api.CapabilityAvailable}, nil
	case api.CapabilityTunMode, api.CapabilityDNSHijack:
		return api.PrepareFeatureReply{Feature: feature, State: api.CapabilityUnavailable, Reason: "not implemented"}, nil
	default:
		return api.PrepareFeatureReply{}, api.NewStructuredError(api.ErrorPlatformFeatureUnsupported, "unsupported", "provider", true)
	}
}

func (f *fakePrivilegedProvider) RunRepairAction(_ context.Context, action string) (api.RunRepairActionReply, *api.StructuredError) {
	f.repairCalls = append(f.repairCalls, action)
	if f.repairErr != nil {
		return api.RunRepairActionReply{}, f.repairErr
	}
	return api.RunRepairActionReply{Action: action, Outcome: "success"}, nil
}

func TestHelloReturnsCapabilityShells(t *testing.T) {
	svc := newTestService(t)
	reply, err := svc.Hello(context.Background(), api.DefaultHelloRequest())
	if err != nil {
		t.Fatal(err)
	}
	if reply.APIVersion != api.APIVersion {
		t.Fatalf("api version = %s", reply.APIVersion)
	}
	if len(reply.RuntimeCapabilities) == 0 || len(reply.PlatformCapabilities) == 0 {
		t.Fatal("expected capability shells")
	}
	for _, capability := range append(reply.RuntimeCapabilities, reply.PlatformCapabilities...) {
		if capability.State == "" {
			t.Fatalf("capability %s has empty state", capability.Name)
		}
	}
}

func TestHelloRejectsUnsupportedAPIVersion(t *testing.T) {
	svc := newTestService(t)
	req := api.DefaultHelloRequest()
	req.ClientAPIVersion = "0"
	_, err := svc.Hello(context.Background(), req)
	if err == nil {
		t.Fatal("expected structured error")
	}
	if err.Code != api.ErrorIPCVersionUnsupported {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestHelloBoundsPrivilegedCapabilityProbe(t *testing.T) {
	privileged := readyFakePrivilegedProvider()
	privileged.statusWaitForCancel = true
	svc := newTestServiceWithPlatform(t, nil, privileged)

	started := time.Now()
	reply, err := svc.Hello(context.Background(), api.DefaultHelloRequest())
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > privilegedCapabilityProbeTimeout+500*time.Millisecond {
		t.Fatalf("hello took %s, want bounded provider probe", elapsed)
	}

	caps := map[string]api.Capability{}
	for _, cap := range reply.PlatformCapabilities {
		caps[cap.Name] = cap
	}
	if caps[api.CapabilityBackgroundService].State != api.CapabilityUnavailable {
		t.Fatalf("background service = %+v", caps[api.CapabilityBackgroundService])
	}
}

func TestProfileCRUD(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// create
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "test-profile",
		Content: `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	// get
	getReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getReply.Profile.Name != "test-profile" {
		t.Fatalf("name = %s", getReply.Profile.Name)
	}
	if getReply.Content == "" {
		t.Fatal("expected content")
	}

	// update
	_, err = svc.UpdateProfileDraft(ctx, api.UpdateProfileDraftRequest{
		ProfileID: pid,
		Content:   `{"inbounds":[],"outbounds":[{"type":"block"}]}`,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// list
	listReply, err := svc.ListProfiles(ctx, api.ListProfilesRequest{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listReply.Profiles) != 1 {
		t.Fatalf("count = %d", len(listReply.Profiles))
	}

	// delete
	_, err = svc.DeleteProfile(ctx, api.DeleteProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	// get after delete
	_, err = svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected error after delete")
	}
	if err.Code != api.ErrorProfileNotFound {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestSnapshotLifecycle(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// create profile
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "snap-test",
		Content: `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	pid := createReply.Profile.ID

	// validate
	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "valid" {
		t.Fatalf("status = %s", validReply.Diagnostics.Status)
	}

	// create snapshot
	snapReply, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	sid := snapReply.Snapshot.ID

	// list snapshots
	listReply, err := svc.ListSnapshots(ctx, api.ListSnapshotsRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(listReply.Snapshots) != 1 {
		t.Fatalf("snapshot count = %d", len(listReply.Snapshots))
	}

	// activate
	_, err = svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: sid})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}

	// get active
	activeReply, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("get active profile: %v", err)
	}
	if activeReply.Profile == nil {
		t.Fatal("expected active profile")
	}
	if activeReply.Profile.ID != pid {
		t.Fatalf("active profile id = %s", activeReply.Profile.ID)
	}

	activeSnapReply, err := svc.GetActiveSnapshot(ctx, api.GetActiveSnapshotRequest{})
	if err != nil {
		t.Fatalf("get active snapshot: %v", err)
	}
	if activeSnapReply.Snapshot == nil {
		t.Fatal("expected active snapshot")
	}
	if activeSnapReply.Snapshot.ID != sid {
		t.Fatalf("active snapshot id = %s", activeSnapReply.Snapshot.ID)
	}

	// rollback (same snapshot)
	_, err = svc.RollbackToSnapshot(ctx, api.RollbackToSnapshotRequest{SnapshotID: sid})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
}

func TestActiveSnapshotSwitchesAcrossProfiles(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	first, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "first",
		Content: `{"inbounds":[],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	firstSnap, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: first.Profile.ID})
	if err != nil {
		t.Fatalf("snapshot first: %v", err)
	}

	second, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "second",
		Content: `{"inbounds":[],"outbounds":[{"type":"block"}]}`,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	secondSnap, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: second.Profile.ID})
	if err != nil {
		t.Fatalf("snapshot second: %v", err)
	}

	if _, err = svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: firstSnap.Snapshot.ID}); err != nil {
		t.Fatalf("activate first: %v", err)
	}
	if _, err = svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: secondSnap.Snapshot.ID}); err != nil {
		t.Fatalf("activate second: %v", err)
	}

	activeProfile, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("get active profile: %v", err)
	}
	if activeProfile.Profile == nil || activeProfile.Profile.ID != second.Profile.ID {
		t.Fatalf("active profile = %+v", activeProfile.Profile)
	}

	activeSnapshot, err := svc.GetActiveSnapshot(ctx, api.GetActiveSnapshotRequest{})
	if err != nil {
		t.Fatalf("get active snapshot: %v", err)
	}
	if activeSnapshot.Snapshot == nil || activeSnapshot.Snapshot.ID != secondSnap.Snapshot.ID {
		t.Fatalf("active snapshot = %+v", activeSnapshot.Snapshot)
	}

	profiles, err := svc.ListProfiles(ctx, api.ListProfilesRequest{})
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	activeCount := 0
	for _, profile := range profiles.Profiles {
		if profile.HasActiveSnapshot {
			activeCount++
			if profile.ID != second.Profile.ID {
				t.Fatalf("unexpected active profile summary: %+v", profile)
			}
			if profile.ActiveSnapshotID == nil || *profile.ActiveSnapshotID != secondSnap.Snapshot.ID {
				t.Fatalf("active snapshot id = %+v", profile.ActiveSnapshotID)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("active profile count = %d", activeCount)
	}
}

func TestValidationBlocksInvalidSnapshot(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "bad-profile",
		Content: `not json`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	// validate shows invalid
	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid, got %s", validReply.Diagnostics.Status)
	}

	// snapshot blocked
	_, err = svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestValidationBlocksEmptyObject(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "empty-obj",
		Content: `{}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid for empty object, got %s", validReply.Diagnostics.Status)
	}

	_, err = svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected snapshot blocked for empty object")
	}
	if err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestValidationBlocksNonArrayFields(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "non-array",
		Content: `{"inbounds":"not-an-array","outbounds":123}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	validReply, err := svc.ValidateProfileDraft(ctx, api.ValidateProfileDraftRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid, got %s", validReply.Diagnostics.Status)
	}

	_, err = svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err == nil {
		t.Fatal("expected snapshot blocked")
	}
	if err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("code = %s", err.Code)
	}
}

func TestEngineStartWithoutActiveSnapshot(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.EngineStart(ctx, api.EngineStartRequest{})
	if err == nil || err.Code != api.ErrorEngineNoActiveSnapshot {
		t.Fatalf("expected ENGINE_NO_ACTIVE_SNAPSHOT, got %v", err)
	}
	status, statusErr := svc.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if statusErr != nil {
		t.Fatalf("status: %v", statusErr)
	}
	if status.Status.LastErrorCode != api.ErrorEngineNoActiveSnapshot {
		t.Fatalf("last error = %s", status.Status.LastErrorCode)
	}
}

func TestEngineStartUsesSnapshotContentNotDraft(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	fake := &fakeAdapter{}
	svc.engine.adapterFactory = func() EngineAdapter {
		return fake
	}

	snapshotContent := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"snapshot"}]}`
	draftContent := `{"inbounds":[],"outbounds":[{"type":"block","tag":"draft"}]}`
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "engine-content",
		Content: snapshotContent,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pid := createReply.Profile.ID

	snapReply, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if _, err := svc.UpdateProfileDraft(ctx, api.UpdateProfileDraftRequest{ProfileID: pid, Content: draftContent}); err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: snapReply.Snapshot.ID}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("engine start: %v", err)
	}
	if fake.configJSON != snapshotContent {
		t.Fatalf("engine used %q, want snapshot content %q", fake.configJSON, snapshotContent)
	}
}

func TestEngineStartBlocksActiveSnapshotMutationWhileStarting(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	started := make(chan struct{})
	release := make(chan struct{})
	fake := &fakeAdapter{startedCh: started, releaseStart: release}
	svc.engine.adapterFactory = func() EngineAdapter {
		return fake
	}

	first := createValidSnapshot(t, svc, ctx, "first")
	second := createValidSnapshot(t, svc, ctx, "second")
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: first}); err != nil {
		t.Fatalf("activate first: %v", err)
	}

	done := make(chan *api.StructuredError, 1)
	go func() {
		_, err := svc.EngineStart(ctx, api.EngineStartRequest{})
		done <- err
	}()

	waitFor(t, started)

	activateDone := make(chan *api.StructuredError, 1)
	go func() {
		_, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: second})
		activateDone <- err
	}()

	select {
	case err := <-activateDone:
		t.Fatalf("activate completed during start window: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if err := waitResult(t, done); err != nil {
		t.Fatalf("engine start: %v", err)
	}
	err := <-activateDone
	if err == nil || err.Code != api.ErrorEngineRunning {
		t.Fatalf("expected ENGINE_RUNNING, got %v", err)
	}
}

func TestSystemProxyStatusRequiresActualOwnershipWithoutDeletingRecord(t *testing.T) {
	proxy := &fakeSystemProxy{state: capability.SystemProxyCurrentState{Enabled: true, Addr: "10.0.0.2", Port: 8080}}
	svc := newTestServiceWithProxy(t, proxy)
	snapshot := &capability.SystemProxySnapshot{Raw: json.RawMessage(`{"baseline":"old"}`)}
	if err := saveProxyOwner(svc.db, &proxyOwnerRecord{
		QKBoxOwned: true,
		Snapshot:   snapshot,
		ProxyAddr:  "127.0.0.1",
		ProxyPort:  7890,
		EnabledAt:  1,
	}); err != nil {
		t.Fatal(err)
	}

	reply, structured := svc.PlatformGetSystemProxyStatus(context.Background(), api.GetSystemProxyStatusRequest{})
	if structured != nil {
		t.Fatalf("status: %v", structured)
	}
	if reply.QKBoxOwned {
		t.Fatal("qkbox_owned must be false when OS proxy no longer points to qkbox")
	}
	if reply.Address != "10.0.0.2" || reply.Port != 8080 {
		t.Fatalf("status address = %s:%d", reply.Address, reply.Port)
	}
	record, err := loadProxyOwner(svc.db)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil {
		t.Fatal("status reads must not delete owner record")
	}
	if string(record.Snapshot.Raw) != string(snapshot.Raw) {
		t.Fatalf("snapshot = %s, want preserved baseline %s", record.Snapshot.Raw, snapshot.Raw)
	}
}

func TestSystemProxyEnableAfterLostOwnershipReappliesWithoutResnapshot(t *testing.T) {
	proxy := &fakeSystemProxy{
		state:    capability.SystemProxyCurrentState{Enabled: true, Addr: "10.0.0.2", Port: 8080},
		snapshot: &capability.SystemProxySnapshot{Raw: json.RawMessage(`{"baseline":"user-proxy"}`)},
	}
	svc := newTestServiceWithProxy(t, proxy)
	startEngineForProxyTest(t, svc)

	if err := saveProxyOwner(svc.db, &proxyOwnerRecord{
		QKBoxOwned: true,
		Snapshot:   &capability.SystemProxySnapshot{Raw: json.RawMessage(`{"baseline":"old-qkbox"}`)},
		ProxyAddr:  "127.0.0.1",
		ProxyPort:  8888,
		EnabledAt:  1,
	}); err != nil {
		t.Fatal(err)
	}

	_, structured := svc.PlatformSetSystemProxyEnabled(context.Background(), api.SetSystemProxyEnabledRequest{Enabled: true})
	if structured != nil {
		t.Fatalf("enable: %v", structured)
	}
	if proxy.snapshotCalls != 0 {
		t.Fatalf("snapshot calls = %d, want no new baseline while qkbox owner record exists", proxy.snapshotCalls)
	}
	if proxy.applyCalls != 1 || proxy.appliedPort != 7890 {
		t.Fatalf("apply = %d to %s:%d, want one apply to fresh listener", proxy.applyCalls, proxy.appliedAddr, proxy.appliedPort)
	}
	record, err := loadProxyOwner(svc.db)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil {
		t.Fatal("expected owner record")
	}
	if string(record.Snapshot.Raw) != `{"baseline":"old-qkbox"}` {
		t.Fatalf("snapshot = %s, want original qkbox baseline", record.Snapshot.Raw)
	}
	if record.ProxyPort != 7890 {
		t.Fatalf("record port = %d, want 7890", record.ProxyPort)
	}
}

func TestBestEffortProxyRestoreKeepsRecordWhenCurrentStateFails(t *testing.T) {
	proxy := &fakeSystemProxy{currentErr: errors.New("read failed")}
	svc := newTestServiceWithProxy(t, proxy)
	if err := saveProxyOwner(svc.db, &proxyOwnerRecord{
		QKBoxOwned: true,
		Snapshot:   &capability.SystemProxySnapshot{Raw: json.RawMessage(`{"baseline":"old"}`)},
		ProxyAddr:  "127.0.0.1",
		ProxyPort:  7890,
		EnabledAt:  1,
	}); err != nil {
		t.Fatal(err)
	}

	svc.bestEffortProxyRestore()

	record, err := loadProxyOwner(svc.db)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil {
		t.Fatal("owner record must remain so next startup can retry repair")
	}
}

func TestPlatformCapabilitiesReflectPrivilegedProvider(t *testing.T) {
	privileged := readyFakePrivilegedProvider()
	svc := newTestServiceWithPlatform(t, nil, privileged)

	reply, structured := svc.PlatformGetCapabilities(context.Background(), api.GetPlatformCapabilitiesRequest{})
	if structured != nil {
		t.Fatalf("capabilities: %v", structured)
	}
	states := map[string]api.Capability{}
	for _, cap := range reply.Capabilities {
		states[cap.Name] = cap
	}
	if states[api.CapabilityBackgroundService].State != api.CapabilityAvailable {
		t.Fatalf("background service = %+v", states[api.CapabilityBackgroundService])
	}
	if states[api.CapabilityTunMode].State != api.CapabilityUnavailable {
		t.Fatalf("tun mode = %+v", states[api.CapabilityTunMode])
	}
}

func TestCreateSnapshotStoresRequiredCapabilities(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "tun",
		Content: `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	snapReply, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: createReply.Profile.ID})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(snapReply.Snapshot.RequiredCapabilities) != 1 || snapReply.Snapshot.RequiredCapabilities[0] != api.CapabilityTunMode {
		t.Fatalf("required capabilities = %+v", snapReply.Snapshot.RequiredCapabilities)
	}
}

func TestEngineReloadSwitchesActiveSnapshotAfterTargetStarts(t *testing.T) {
	svc := newTestServiceWithPlatform(t, nil, readyFakePrivilegedProvider())
	ctx := context.Background()
	adapters := installAdapterSequence(svc)

	previous := createSnapshotWithContent(t, svc, ctx, "previous", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "target", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"target"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
	if structured != nil {
		t.Fatalf("reload: %v", structured)
	}
	if reply.Outcome != api.ReloadOutcomeSuccess {
		t.Fatalf("outcome = %s", reply.Outcome)
	}
	if reply.ActiveSnapshotID != target {
		t.Fatalf("active in reply = %s, want %s", reply.ActiveSnapshotID, target)
	}
	active, err := svc.GetActiveSnapshot(ctx, api.GetActiveSnapshotRequest{})
	if err != nil {
		t.Fatalf("active: %v", err)
	}
	if active.Snapshot == nil || active.Snapshot.ID != target {
		t.Fatalf("active snapshot = %+v", active.Snapshot)
	}
	if len(*adapters) < 2 || (*adapters)[1].configJSON == "" || (*adapters)[1].configJSON == (*adapters)[0].configJSON {
		t.Fatalf("adapter configs = %+v", *adapters)
	}
}

func TestEngineReloadTargetFailureRollsBackPreviousRuntime(t *testing.T) {
	svc := newTestServiceWithPlatform(t, nil, readyFakePrivilegedProvider())
	ctx := context.Background()
	startErrs := []error{nil, errors.New("target failed"), nil}
	installAdapterSequenceWithStartErrors(svc, startErrs)

	previous := createSnapshotWithContent(t, svc, ctx, "previous", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "target", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"target"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
	if structured != nil {
		t.Fatalf("reload: %v", structured)
	}
	if reply.Outcome != api.ReloadOutcomeRolledBack {
		t.Fatalf("outcome = %s", reply.Outcome)
	}
	if reply.ActiveSnapshotID != previous {
		t.Fatalf("active = %s, want previous %s", reply.ActiveSnapshotID, previous)
	}
	status, err := svc.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status.State != "STARTED" || status.Status.ActiveSnapshotID != previous {
		t.Fatalf("status = %+v", status.Status)
	}
}

func TestEngineReloadRollbackUsesPreparedStartPipeline(t *testing.T) {
	privileged := readyFakePrivilegedProvider()
	privileged.prepare = map[string]api.PrepareFeatureReply{
		api.CapabilityTunMode: {Feature: api.CapabilityTunMode, State: api.CapabilityAvailable},
	}
	svc := newTestServiceWithPlatform(t, nil, privileged)
	ctx := context.Background()
	startErrs := []error{nil, errors.New("target failed"), nil}
	installAdapterSequenceWithStartErrors(svc, startErrs)

	previous := createSnapshotWithContent(t, svc, ctx, "tun-previous", `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "target", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"target"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
	if structured != nil {
		t.Fatalf("reload: %v", structured)
	}
	if reply.Outcome != api.ReloadOutcomeRolledBack {
		t.Fatalf("outcome = %s", reply.Outcome)
	}

	tunPrepareCount := 0
	for _, feature := range privileged.prepared {
		if feature == api.CapabilityTunMode {
			tunPrepareCount++
		}
	}
	if tunPrepareCount != 2 {
		t.Fatalf("tun prepare count = %d, want initial start + rollback prepare; calls=%+v", tunPrepareCount, privileged.prepared)
	}
}

func TestEngineReloadTargetAndRollbackFailureReportsDegraded(t *testing.T) {
	svc := newTestServiceWithPlatform(t, nil, readyFakePrivilegedProvider())
	ctx := context.Background()
	startErrs := []error{nil, errors.New("target failed"), errors.New("rollback failed")}
	installAdapterSequenceWithStartErrors(svc, startErrs)

	previous := createSnapshotWithContent(t, svc, ctx, "previous", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "target", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"target"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
	if structured != nil {
		t.Fatalf("reload: %v", structured)
	}
	if reply.Outcome != api.ReloadOutcomeDegraded {
		t.Fatalf("outcome = %s", reply.Outcome)
	}
}

func TestEngineReloadPlatformPrepareFailureDoesNotStopRuntime(t *testing.T) {
	privileged := readyFakePrivilegedProvider()
	svc := newTestServiceWithPlatform(t, nil, privileged)
	ctx := context.Background()
	installAdapterSequence(svc)

	previous := createSnapshotWithContent(t, svc, ctx, "previous", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "tun-target", `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct","tag":"target"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
	if structured != nil {
		t.Fatalf("reload: %v", structured)
	}
	if reply.Outcome != api.ReloadOutcomeFailedPlatformPrepare {
		t.Fatalf("outcome = %s", reply.Outcome)
	}
	status, err := svc.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status.State != "STARTED" || status.Status.ActiveSnapshotID != previous {
		t.Fatalf("status = %+v", status.Status)
	}
	if len(privileged.prepared) != 1 || privileged.prepared[0] != api.CapabilityTunMode {
		t.Fatalf("prepared = %+v", privileged.prepared)
	}
}

func TestEngineReloadCleanupFailureDoesNotStopRuntime(t *testing.T) {
	proxy := &fakeSystemProxy{
		state:      capability.SystemProxyCurrentState{Enabled: true, Addr: "127.0.0.1", Port: 7890},
		restoreErr: errors.New("restore failed"),
	}
	svc := newTestServiceWithPlatform(t, proxy, readyFakePrivilegedProvider())
	ctx := context.Background()
	installAdapterSequence(svc)
	previous := createSnapshotWithContent(t, svc, ctx, "previous", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "target", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"target"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := saveProxyOwner(svc.db, &proxyOwnerRecord{
		QKBoxOwned: true,
		Snapshot:   &capability.SystemProxySnapshot{Raw: json.RawMessage(`{"baseline":"old"}`)},
		ProxyAddr:  "127.0.0.1",
		ProxyPort:  7890,
		EnabledAt:  1,
	}); err != nil {
		t.Fatal(err)
	}

	reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
	if structured != nil {
		t.Fatalf("reload: %v", structured)
	}
	if reply.Outcome != api.ReloadOutcomeCleanupFailed {
		t.Fatalf("outcome = %s", reply.Outcome)
	}
	status, err := svc.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Status.State != "STARTED" || status.Status.ActiveSnapshotID != previous {
		t.Fatalf("status = %+v", status.Status)
	}
}

func TestEngineReloadSerializesActiveSnapshotMutation(t *testing.T) {
	svc := newTestServiceWithPlatform(t, nil, readyFakePrivilegedProvider())
	ctx := context.Background()

	targetStarted := make(chan struct{})
	releaseTarget := make(chan struct{})
	var adapters []*fakeAdapter
	svc.engine.adapterFactory = func() EngineAdapter {
		adapter := &fakeAdapter{}
		if len(adapters) == 1 {
			adapter.startedCh = targetStarted
			adapter.releaseStart = releaseTarget
		}
		adapters = append(adapters, adapter)
		return adapter
	}

	previous := createSnapshotWithContent(t, svc, ctx, "previous", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"previous"}]}`)
	target := createSnapshotWithContent(t, svc, ctx, "target", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"target"}]}`)
	other := createSnapshotWithContent(t, svc, ctx, "other", `{"inbounds":[],"outbounds":[{"type":"direct","tag":"other"}]}`)
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: previous}); err != nil {
		t.Fatalf("activate previous: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	reloadDone := make(chan api.EngineReloadReply, 1)
	go func() {
		reply, structured := svc.EngineReload(ctx, api.EngineReloadRequest{SnapshotID: target})
		if structured != nil {
			t.Errorf("reload structured error: %v", structured)
		}
		reloadDone <- reply
	}()

	select {
	case <-targetStarted:
	case <-time.After(time.Second):
		t.Fatal("target runtime did not begin starting")
	}

	activateDone := make(chan *api.StructuredError, 1)
	go func() {
		_, structured := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: other})
		activateDone <- structured
	}()

	select {
	case err := <-activateDone:
		t.Fatalf("activate completed during reload window: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseTarget)
	reply := <-reloadDone
	if reply.Outcome != api.ReloadOutcomeSuccess {
		t.Fatalf("reload outcome = %s", reply.Outcome)
	}
	err := <-activateDone
	if err == nil || err.Code != api.ErrorEngineRunning {
		t.Fatalf("activate error = %+v, want ENGINE_RUNNING", err)
	}
	active, activeErr := svc.GetActiveSnapshot(ctx, api.GetActiveSnapshotRequest{})
	if activeErr != nil {
		t.Fatalf("active: %v", activeErr)
	}
	if active.Snapshot == nil || active.Snapshot.ID != target {
		t.Fatalf("active snapshot = %+v, want target", active.Snapshot)
	}
}

func createValidSnapshot(t *testing.T, svc *Service, ctx context.Context, name string) string {
	t.Helper()
	return createSnapshotWithContent(t, svc, ctx, name, `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)
}

func createSnapshotWithContent(t *testing.T, svc *Service, ctx context.Context, name, content string) string {
	t.Helper()
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    name,
		Content: content,
	})
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	snapReply, err := svc.CreateProfileSnapshot(ctx, api.CreateProfileSnapshotRequest{ProfileID: createReply.Profile.ID})
	if err != nil {
		t.Fatalf("snapshot %s: %v", name, err)
	}
	return snapReply.Snapshot.ID
}

func installAdapterSequence(svc *Service) *[]*fakeAdapter {
	return installAdapterSequenceWithStartErrors(svc, nil)
}

func installAdapterSequenceWithStartErrors(svc *Service, startErrs []error) *[]*fakeAdapter {
	var adapters []*fakeAdapter
	svc.engine.adapterFactory = func() EngineAdapter {
		adapter := &fakeAdapter{}
		if len(startErrs) > len(adapters) {
			adapter.startErr = startErrs[len(adapters)]
		}
		adapters = append(adapters, adapter)
		return adapter
	}
	return &adapters
}

func startEngineForProxyTest(t *testing.T, svc *Service) {
	t.Helper()
	ctx := context.Background()
	fake := &fakeAdapter{}
	svc.engine.adapterFactory = func() EngineAdapter {
		return fake
	}
	snapshotID := createValidSnapshot(t, svc, ctx, "proxy-engine")
	if _, err := svc.ActivateProfileSnapshot(ctx, api.ActivateProfileSnapshotRequest{SnapshotID: snapshotID}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("engine start: %v", err)
	}
}

func TestContentIsEncrypted(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	content := `{"secret":"my-password-123"}`
	_, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "encrypted-test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// retrieve encrypted content via repo and verify it's not plaintext
	contents, listErr := svc.db.ListAllContent()
	if listErr != nil {
		t.Fatalf("list content: %v", listErr)
	}
	if len(contents) == 0 {
		t.Fatal("no content stored")
	}
	for _, c := range contents {
		raw := string(c.Ciphertext)
		if raw == content {
			t.Fatal("content stored in plaintext")
		}
		if len(c.Ciphertext) == 0 {
			t.Fatal("empty ciphertext")
		}
		if len(c.IV) == 0 {
			t.Fatal("empty IV")
		}
	}

	// verify we can still decrypt and get the original content
	getReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: contents[0].SourceID})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getReply.Content != content {
		t.Fatalf("decrypted content mismatch: got %q", getReply.Content)
	}
}
