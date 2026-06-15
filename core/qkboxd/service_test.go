package qkboxd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/internal/persistence"
	"github.com/zclkkk/qkbox/internal/provideripc"
	"github.com/zclkkk/qkbox/internal/runtimeapi"
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
	return newTestServiceWithPlatformAndExtension(t, proxy, privileged, nil)
}

func newTestServiceWithPlatformAndExtension(t *testing.T, proxy capability.SystemProxyProvider, privileged capability.PrivilegedProvider, extension capability.NetworkExtensionRuntime) *Service {
	t.Helper()
	dir := t.TempDir()
	db, err := persistence.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	svc := NewServiceWithNetworkExtension(context.Background(), db, proxy, privileged, extension)
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
	runtimeStartErr     *api.StructuredError
	runtimeStarts       []provideripc.RuntimeStartRequest
	runtimeStops        []provideripc.RuntimeStopRequest
	runtimeHeartbeats   []provideripc.RuntimeHeartbeatRequest
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

func (f *fakePrivilegedProvider) RuntimeStart(_ context.Context, req provideripc.RuntimeStartRequest) (provideripc.RuntimeStartReply, *api.StructuredError) {
	f.runtimeStarts = append(f.runtimeStarts, req)
	if f.runtimeStartErr != nil {
		return provideripc.RuntimeStartReply{}, f.runtimeStartErr
	}
	return provideripc.RuntimeStartReply{OwnerState: api.ProviderOwnerState{
		Owned:     true,
		SessionID: req.SessionID,
		RuntimeID: req.RuntimeID,
		ProfileID: req.ProfileID,
		Mode:      req.Mode,
	}}, nil
}

func (f *fakePrivilegedProvider) RuntimeStop(_ context.Context, req provideripc.RuntimeStopRequest) (provideripc.RuntimeStopReply, *api.StructuredError) {
	f.runtimeStops = append(f.runtimeStops, req)
	return provideripc.RuntimeStopReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeHeartbeat(_ context.Context, req provideripc.RuntimeHeartbeatRequest) (provideripc.RuntimeHeartbeatReply, *api.StructuredError) {
	f.runtimeHeartbeats = append(f.runtimeHeartbeats, req)
	return provideripc.RuntimeHeartbeatReply{OwnerState: api.ProviderOwnerState{Owned: true, SessionID: req.SessionID, RuntimeID: req.RuntimeID}}, nil
}

func (f *fakePrivilegedProvider) RuntimeGetStatus(context.Context, provideripc.RuntimeGetStatusRequest) (provideripc.RuntimeGetStatusReply, *api.StructuredError) {
	return provideripc.RuntimeGetStatusReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeGetRuntimeCapabilities(context.Context, provideripc.RuntimeGetRuntimeCapabilitiesRequest) (provideripc.RuntimeGetRuntimeCapabilitiesReply, *api.StructuredError) {
	return provideripc.RuntimeGetRuntimeCapabilitiesReply{Capabilities: api.RuntimeCapabilityShell()}, nil
}

func (f *fakePrivilegedProvider) RuntimeGetTraffic(context.Context, provideripc.RuntimeGetTrafficRequest) (provideripc.RuntimeGetTrafficReply, *api.StructuredError) {
	return provideripc.RuntimeGetTrafficReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeGetConnections(context.Context, provideripc.RuntimeGetConnectionsRequest) (provideripc.RuntimeGetConnectionsReply, *api.StructuredError) {
	return provideripc.RuntimeGetConnectionsReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeListGroups(context.Context, provideripc.RuntimeListGroupsRequest) (provideripc.RuntimeListGroupsReply, *api.StructuredError) {
	return provideripc.RuntimeListGroupsReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeSelectOutbound(context.Context, provideripc.RuntimeSelectOutboundRequest) (provideripc.RuntimeSelectOutboundReply, *api.StructuredError) {
	return provideripc.RuntimeSelectOutboundReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeURLTest(context.Context, provideripc.RuntimeURLTestRequest) (provideripc.RuntimeURLTestReply, *api.StructuredError) {
	return provideripc.RuntimeURLTestReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeCloseConnection(context.Context, provideripc.RuntimeCloseConnectionRequest) (provideripc.RuntimeCloseConnectionReply, *api.StructuredError) {
	return provideripc.RuntimeCloseConnectionReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeCloseAllConnections(context.Context, provideripc.RuntimeCloseAllConnectionsRequest) (provideripc.RuntimeCloseAllConnectionsReply, *api.StructuredError) {
	return provideripc.RuntimeCloseAllConnectionsReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeListenerInfo(context.Context, provideripc.RuntimeListenerInfoRequest) (provideripc.RuntimeListenerInfoReply, *api.StructuredError) {
	return provideripc.RuntimeListenerInfoReply{}, nil
}

func (f *fakePrivilegedProvider) RuntimeSubscribeEvents(ctx context.Context, _ provideripc.RuntimeSubscribeEventsRequest) (<-chan provideripc.EventFrame, *api.StructuredError) {
	ch := make(chan provideripc.EventFrame)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

type fakeNetworkExtensionRuntime struct {
	status api.NetworkExtensionStatus
	starts []capability.NetworkExtensionStartRequest
	stops  []capability.NetworkExtensionStopRequest
}

func (f *fakeNetworkExtensionRuntime) Status(context.Context) api.NetworkExtensionStatus {
	return f.status
}

func (f *fakeNetworkExtensionRuntime) Start(_ context.Context, req capability.NetworkExtensionStartRequest) (capability.NetworkExtensionStartReply, *api.StructuredError) {
	f.starts = append(f.starts, req)
	return capability.NetworkExtensionStartReply{}, nil
}

func (f *fakeNetworkExtensionRuntime) Stop(_ context.Context, req capability.NetworkExtensionStopRequest) (capability.NetworkExtensionStopReply, *api.StructuredError) {
	f.stops = append(f.stops, req)
	return capability.NetworkExtensionStopReply{}, nil
}

func (f *fakeNetworkExtensionRuntime) Heartbeat(context.Context, capability.NetworkExtensionHeartbeatRequest) (capability.NetworkExtensionHeartbeatReply, *api.StructuredError) {
	return capability.NetworkExtensionHeartbeatReply{}, nil
}

func (f *fakeNetworkExtensionRuntime) RuntimeCapabilities(context.Context, capability.NetworkExtensionRuntimeRequest) ([]api.Capability, *api.StructuredError) {
	return api.RuntimeCapabilityShell(), nil
}

func (f *fakeNetworkExtensionRuntime) TrafficSnapshot(context.Context, capability.NetworkExtensionRuntimeRequest) (api.TrafficSnapshot, *api.StructuredError) {
	return api.TrafficSnapshot{}, nil
}

func (f *fakeNetworkExtensionRuntime) ConnectionSnapshot(context.Context, capability.NetworkExtensionRuntimeRequest) (api.ConnectionSnapshot, *api.StructuredError) {
	return api.ConnectionSnapshot{}, nil
}

func (f *fakeNetworkExtensionRuntime) ListGroups(context.Context, capability.NetworkExtensionRuntimeRequest) ([]api.OutboundGroup, *api.StructuredError) {
	return nil, nil
}

func (f *fakeNetworkExtensionRuntime) SelectOutbound(context.Context, capability.NetworkExtensionSelectOutboundRequest) (api.OutboundGroup, *api.StructuredError) {
	return api.OutboundGroup{}, nil
}

func (f *fakeNetworkExtensionRuntime) URLTest(context.Context, capability.NetworkExtensionURLTestRequest) ([]api.URLTestResult, *api.StructuredError) {
	return nil, nil
}

func (f *fakeNetworkExtensionRuntime) CloseConnection(context.Context, capability.NetworkExtensionCloseConnectionRequest) *api.StructuredError {
	return nil
}

func (f *fakeNetworkExtensionRuntime) CloseAllConnections(context.Context, capability.NetworkExtensionRuntimeRequest) *api.StructuredError {
	return nil
}

func (f *fakeNetworkExtensionRuntime) ListenerInfo(context.Context, capability.NetworkExtensionRuntimeRequest) ([]runtimeapi.ListenerInfo, *api.StructuredError) {
	return nil, nil
}

func (f *fakeNetworkExtensionRuntime) SubscribeEvents(ctx context.Context, _ capability.NetworkExtensionRuntimeRequest) (<-chan api.RuntimeEvent, *api.StructuredError) {
	ch := make(chan api.RuntimeEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
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

func TestDarwinPrepareUsesNetworkExtension(t *testing.T) {
	oldGOOS := runtimeGOOS
	runtimeGOOS = "darwin"
	t.Cleanup(func() { runtimeGOOS = oldGOOS })

	privileged := readyFakePrivilegedProvider()
	extension := &fakeNetworkExtensionRuntime{
		status: api.NetworkExtensionStatus{
			Installed:  true,
			Reachable:  true,
			Authorized: true,
			Capabilities: []api.Capability{
				{Name: api.CapabilityTunMode, State: api.CapabilityAvailable},
			},
		},
	}
	svc := newTestServiceWithPlatformAndExtension(t, nil, privileged, extension)

	reply, structured := svc.PlatformPrepareFeature(context.Background(), api.PrepareFeatureRequest{Feature: api.CapabilityTunMode})
	if structured != nil {
		t.Fatal(structured)
	}
	if reply.State != api.CapabilityAvailable {
		t.Fatalf("state = %s, want available", reply.State)
	}
	if len(privileged.prepared) != 0 {
		t.Fatalf("privileged prepare called on Darwin: %+v", privileged.prepared)
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

	// update metadata
	updateReply, err := svc.UpdateProfile(ctx, api.UpdateProfileRequest{
		ProfileID: pid,
		Name:      "renamed-profile",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateReply.Profile.Name != "renamed-profile" {
		t.Fatalf("updated name = %s", updateReply.Profile.Name)
	}

	// save content
	_, err = svc.SaveProfileContent(ctx, api.SaveProfileContentRequest{
		ProfileID: pid,
		Content:   `{"inbounds":[],"outbounds":[{"type":"block"}]}`,
	})
	if err != nil {
		t.Fatalf("save content: %v", err)
	}

	validateReply, err := svc.ValidateProfileContent(ctx, api.ValidateProfileContentRequest{
		ProfileID: pid,
		Content:   `{"inbounds":[],"outbounds":[{"type":"block"}]}`,
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validateReply.Diagnostics.Status != "valid" {
		t.Fatalf("diagnostics = %+v", validateReply.Diagnostics)
	}

	activeReply, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if activeReply.Profile.ID != pid {
		t.Fatalf("active profile = %+v", activeReply.Profile)
	}
	getActiveReply, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if getActiveReply.Profile == nil || getActiveReply.Profile.ID != pid {
		t.Fatalf("get active profile = %+v", getActiveReply.Profile)
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

func TestProfileSubscriptionRefreshUpdatesProfileContent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	initial := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"old"}]}`
	updated := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"new"}]}`
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{Name: "remote", Content: initial})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: createReply.Profile.ID}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(updated))
	}))
	defer server.Close()

	subReply, err := svc.CreateProfileSubscription(ctx, api.CreateProfileSubscriptionRequest{
		ProfileID: createReply.Profile.ID,
		Name:      "remote source",
		URL:       server.URL,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	refreshReply, err := svc.RefreshProfileSubscription(ctx, api.RefreshProfileSubscriptionRequest{SubscriptionID: subReply.Subscription.ID})
	if err != nil {
		t.Fatalf("refresh subscription: %v", err)
	}
	if refreshReply.Diagnostics.Status != "valid" {
		t.Fatalf("diagnostics = %+v", refreshReply.Diagnostics)
	}
	if refreshReply.Subscription.ContentSHA256 == "" || refreshReply.Subscription.LastStatus != "updated" {
		t.Fatalf("subscription = %+v", refreshReply.Subscription)
	}

	profileReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: createReply.Profile.ID})
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profileReply.Content != updated {
		t.Fatalf("profile content = %s", profileReply.Content)
	}
	activeReply, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("active profile: %v", err)
	}
	if activeReply.Profile == nil || activeReply.Profile.ID != createReply.Profile.ID {
		t.Fatalf("active profile changed: %+v", activeReply.Profile)
	}
}

func TestInvalidProfileSubscriptionRefreshDoesNotReplaceContent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	initial := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"old"}]}`
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{Name: "remote", Content: initial})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	subReply, err := svc.CreateProfileSubscription(ctx, api.CreateProfileSubscriptionRequest{
		ProfileID: createReply.Profile.ID,
		Name:      "remote source",
		URL:       server.URL,
	})
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	_, err = svc.RefreshProfileSubscription(ctx, api.RefreshProfileSubscriptionRequest{SubscriptionID: subReply.Subscription.ID})
	if err == nil || err.Code != api.ErrorConfigValidationFailed {
		t.Fatalf("expected validation failure, got %v", err)
	}

	profileReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: createReply.Profile.ID})
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profileReply.Content != initial {
		t.Fatalf("profile content changed: %s", profileReply.Content)
	}
	subsReply, err := svc.ListProfileSubscriptions(ctx, api.ListProfileSubscriptionsRequest{ProfileID: createReply.Profile.ID})
	if err != nil {
		t.Fatalf("list subscriptions: %v", err)
	}
	if len(subsReply.Subscriptions) != 1 || subsReply.Subscriptions[0].LastStatus != "failed" || subsReply.Subscriptions[0].LastErrorCode != api.ErrorConfigValidationFailed {
		t.Fatalf("subscription state = %+v", subsReply.Subscriptions)
	}
}

func TestDataAssetRefreshWritesContentAddressedCache(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	content := []byte(`{"rules":[{"domain_suffix":["example.com"]}]}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"asset-v1"`)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	createReply, err := svc.CreateDataAsset(ctx, api.CreateDataAssetRequest{
		Kind:      "rule_set",
		Name:      "rules",
		SourceURL: server.URL,
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	refreshReply, err := svc.RefreshDataAsset(ctx, api.RefreshDataAssetRequest{AssetID: createReply.Asset.ID})
	if err != nil {
		t.Fatalf("refresh asset: %v", err)
	}
	if refreshReply.Asset.Status != "available" || refreshReply.Asset.CacheKey == "" || refreshReply.Asset.ContentSHA256 == "" {
		t.Fatalf("asset = %+v", refreshReply.Asset)
	}
	if refreshReply.Asset.Version != `"asset-v1"` {
		t.Fatalf("version = %s", refreshReply.Asset.Version)
	}
	cachePath := filepath.Join(svc.db.StateDir(), "assets", filepath.FromSlash(refreshReply.Asset.CacheKey))
	if _, statErr := os.Stat(cachePath); statErr != nil {
		t.Fatalf("cache file missing: %v", statErr)
	}
}

func TestDiagnosticsReportRedactsRemoteURLs(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "diag",
		Content: `{"inbounds":[],"outbounds":[{"type":"direct"}]}`,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if _, err := svc.CreateProfileSubscription(ctx, api.CreateProfileSubscriptionRequest{
		ProfileID: createReply.Profile.ID,
		Name:      "secret subscription",
		URL:       "https://user:secret@example.com/sub.json?token=abc123",
	}); err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	if _, err := svc.CreateDataAsset(ctx, api.CreateDataAssetRequest{
		Kind:      "rule_set",
		Name:      "secret asset",
		SourceURL: "https://example.com/token-in-path/rules.json?access_token=abc123",
	}); err != nil {
		t.Fatalf("create asset: %v", err)
	}

	reply, err := svc.DiagnosticsGetReport(ctx, api.GetDiagnosticsReportRequest{})
	if err != nil {
		t.Fatalf("diagnostics report: %v", err)
	}
	payload, marshalErr := json.Marshal(reply.Report)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if strings.Contains(string(payload), "abc123") || strings.Contains(string(payload), "user:secret") || strings.Contains(string(payload), "token-in-path") {
		t.Fatalf("diagnostics leaked URL secret: %s", payload)
	}
	if !strings.Contains(string(payload), "redacted=1") {
		t.Fatalf("diagnostics missing redacted URL marker: %s", payload)
	}
}

func TestDebugBundleDoesNotIncludeProfileContentOrURLSecrets(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "bundle",
		Content: `{"inbounds":[],"outbounds":[{"type":"direct"}],"password":"super-secret"}`,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if _, err := svc.CreateProfileSubscription(ctx, api.CreateProfileSubscriptionRequest{
		ProfileID: createReply.Profile.ID,
		Name:      "secret subscription",
		URL:       "https://user:secret@example.com/private/sub.json?token=abc123#frag",
	}); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	reply, err := svc.DiagnosticsCreateDebugBundle(ctx, api.CreateDebugBundleRequest{})
	if err != nil {
		t.Fatalf("debug bundle: %v", err)
	}
	reader, openErr := zip.OpenReader(reply.BundlePath)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer reader.Close()

	var bundle strings.Builder
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		payload, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		bundle.WriteString(file.Name)
		bundle.Write(payload)
	}
	content := bundle.String()
	for _, forbidden := range []string{"super-secret", "abc123", "user:secret", "private/sub.json", "#frag", "encrypted_content", "proxy_owner"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("debug bundle leaked %q in %s", forbidden, content)
		}
	}
	for _, required := range []string{"manifest.json", "diagnostics.json", "README.txt"} {
		if !strings.Contains(content, required) {
			t.Fatalf("debug bundle missing %s", required)
		}
	}
	for _, required := range []string{"api_version", "schema_revision", "platform", "URL paths"} {
		if !strings.Contains(content, required) {
			t.Fatalf("debug bundle missing manifest/redaction marker %s", required)
		}
	}
}

func TestProfileActivationLifecycle(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	content := `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`

	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    "profile-test",
		Content: content,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	pid := createReply.Profile.ID

	validReply, err := svc.ValidateProfileContent(ctx, api.ValidateProfileContentRequest{ProfileID: pid, Content: content})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "valid" {
		t.Fatalf("status = %s", validReply.Diagnostics.Status)
	}

	updated, err := svc.UpdateProfile(ctx, api.UpdateProfileRequest{ProfileID: pid, Name: "renamed"})
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Profile.Name != "renamed" {
		t.Fatalf("name = %s", updated.Profile.Name)
	}

	savedContent := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"saved"}]}`
	if _, err := svc.SaveProfileContent(ctx, api.SaveProfileContentRequest{ProfileID: pid, Content: savedContent}); err != nil {
		t.Fatalf("save content: %v", err)
	}

	getReply, err := svc.GetProfile(ctx, api.GetProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if getReply.Content != savedContent {
		t.Fatalf("content = %q", getReply.Content)
	}

	activeReply, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: pid})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if activeReply.Profile.ID != pid {
		t.Fatalf("active reply profile id = %s", activeReply.Profile.ID)
	}

	getActiveReply, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("get active profile: %v", err)
	}
	if getActiveReply.Profile == nil || getActiveReply.Profile.ID != pid {
		t.Fatalf("active profile = %+v", getActiveReply.Profile)
	}
}

func TestActiveProfileSwitchesAcrossProfiles(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	first := createProfileWithContent(t, svc, ctx, "first", `{"inbounds":[],"outbounds":[{"type":"direct"}]}`)
	second := createProfileWithContent(t, svc, ctx, "second", `{"inbounds":[],"outbounds":[{"type":"block"}]}`)

	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: first}); err != nil {
		t.Fatalf("activate first: %v", err)
	}
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: second}); err != nil {
		t.Fatalf("activate second: %v", err)
	}

	activeProfile, err := svc.GetActiveProfile(ctx, api.GetActiveProfileRequest{})
	if err != nil {
		t.Fatalf("get active profile: %v", err)
	}
	if activeProfile.Profile == nil || activeProfile.Profile.ID != second {
		t.Fatalf("active profile = %+v", activeProfile.Profile)
	}

	profiles, err := svc.ListProfiles(ctx, api.ListProfilesRequest{})
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles.Profiles) != 2 {
		t.Fatalf("profile count = %d", len(profiles.Profiles))
	}
}

func TestValidateProfileContentReportsInvalidJSON(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	profileID := createProfileWithContent(t, svc, ctx, "bad-profile", `not json`)

	validReply, err := svc.ValidateProfileContent(ctx, api.ValidateProfileContentRequest{ProfileID: profileID, Content: `not json`})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid, got %s", validReply.Diagnostics.Status)
	}
}

func TestValidateProfileContentReportsEmptyObject(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	profileID := createProfileWithContent(t, svc, ctx, "empty-obj", `{}`)

	validReply, err := svc.ValidateProfileContent(ctx, api.ValidateProfileContentRequest{ProfileID: profileID, Content: `{}`})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid for empty object, got %s", validReply.Diagnostics.Status)
	}
}

func TestValidateProfileContentReportsNonArrayFields(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	content := `{"inbounds":"not-an-array","outbounds":123}`
	profileID := createProfileWithContent(t, svc, ctx, "non-array", content)

	validReply, err := svc.ValidateProfileContent(ctx, api.ValidateProfileContentRequest{ProfileID: profileID, Content: content})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if validReply.Diagnostics.Status != "invalid" {
		t.Fatalf("expected invalid, got %s", validReply.Diagnostics.Status)
	}
}

func TestEngineStartWithoutActiveProfile(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.EngineStart(ctx, api.EngineStartRequest{})
	if err == nil || err.Code != api.ErrorEngineNoActiveProfile {
		t.Fatalf("expected ENGINE_NO_ACTIVE_PROFILE, got %v", err)
	}
	status, statusErr := svc.EngineGetStatus(ctx, api.EngineGetStatusRequest{})
	if statusErr != nil {
		t.Fatalf("status: %v", statusErr)
	}
	if status.Status.LastErrorCode != api.ErrorEngineNoActiveProfile {
		t.Fatalf("last error = %s", status.Status.LastErrorCode)
	}
}

func TestEngineStartUsesActiveProfileContent(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	fake := &fakeRuntimeOwner{}
	svc.engine.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	initialContent := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"initial"}]}`
	activeContent := `{"inbounds":[],"outbounds":[{"type":"block","tag":"active"}]}`
	profileID := createProfileWithContent(t, svc, ctx, "engine-content", initialContent)
	if _, err := svc.SaveProfileContent(ctx, api.SaveProfileContentRequest{ProfileID: profileID, Content: activeContent}); err != nil {
		t.Fatalf("save content: %v", err)
	}
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: profileID}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("engine start: %v", err)
	}
	if fake.target.ProfileID != profileID {
		t.Fatalf("target profile = %s, want %s", fake.target.ProfileID, profileID)
	}
	if fake.configJSON != activeContent {
		t.Fatalf("engine used %q, want profile content %q", fake.configJSON, activeContent)
	}
}

func TestEngineStartBlocksActiveProfileMutationWhileStarting(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	started := make(chan struct{})
	release := make(chan struct{})
	fake := &fakeRuntimeOwner{startedCh: started, releaseStart: release}
	svc.engine.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}

	first := createValidProfile(t, svc, ctx, "first")
	second := createValidProfile(t, svc, ctx, "second")
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: first}); err != nil {
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
		_, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: second})
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

func TestDarwinPlatformCapabilitiesReflectNetworkExtension(t *testing.T) {
	oldGOOS := runtimeGOOS
	runtimeGOOS = "darwin"
	t.Cleanup(func() { runtimeGOOS = oldGOOS })

	extension := &fakeNetworkExtensionRuntime{
		status: api.NetworkExtensionStatus{
			Installed:  true,
			Reachable:  true,
			Authorized: true,
			Capabilities: []api.Capability{
				{Name: api.CapabilityTunMode, State: api.CapabilityAvailable},
				{Name: api.CapabilityDNSHijack, State: api.CapabilityAvailable},
			},
		},
	}
	svc := newTestServiceWithPlatformAndExtension(t, nil, readyFakePrivilegedProvider(), extension)

	reply, structured := svc.PlatformGetCapabilities(context.Background(), api.GetPlatformCapabilitiesRequest{})
	if structured != nil {
		t.Fatalf("capabilities: %v", structured)
	}
	states := map[string]api.Capability{}
	for _, cap := range reply.Capabilities {
		states[cap.Name] = cap
	}
	if states[api.CapabilityTunMode].State != api.CapabilityAvailable {
		t.Fatalf("tun mode = %+v", states[api.CapabilityTunMode])
	}
	if states[api.CapabilityBackgroundService].State != api.CapabilityUnsupported {
		t.Fatalf("background service = %+v", states[api.CapabilityBackgroundService])
	}
}

func TestLoadRuntimeStartTargetCarriesRequiredCapabilities(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	profileID := createProfileWithContent(t, svc, ctx, "tun-target", `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct"}]}`)

	target, err := svc.loadRuntimeStartTargetByID(profileID)
	if err != nil {
		t.Fatalf("load target: %v", err)
	}
	if target.ProfileID != profileID {
		t.Fatalf("profile id = %s, want %s", target.ProfileID, profileID)
	}
	if len(target.RequiredCapabilities) != 1 || target.RequiredCapabilities[0] != api.CapabilityTunMode {
		t.Fatalf("required capabilities = %+v", target.RequiredCapabilities)
	}
}

func TestEngineStartUsesProviderHostedOwnerForMachineRuntime(t *testing.T) {
	if !supportsProviderHostedMachineRuntime() {
		t.Skip("provider-hosted machine runtime selection is only available on Windows and Linux")
	}
	privileged := readyFakePrivilegedProvider()
	privileged.prepare = map[string]api.PrepareFeatureReply{
		api.CapabilityTunMode: {Feature: api.CapabilityTunMode, State: api.CapabilityAvailable},
	}
	svc := newTestServiceWithPlatform(t, nil, privileged)
	ctx := context.Background()
	profileID := createProfileWithContent(t, svc, ctx, "tun-target", `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: profileID}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(privileged.runtimeStarts) != 1 {
		t.Fatalf("runtime starts = %d, want provider-hosted start", len(privileged.runtimeStarts))
	}
	start := privileged.runtimeStarts[0]
	if start.Mode != api.RuntimeModeMachineNetwork || start.ProfileID != profileID {
		t.Fatalf("runtime start = %+v", start)
	}
	if start.ConfigJSON == "" {
		t.Fatal("provider runtime start must receive config JSON in memory")
	}
	if len(start.RequiredCapabilities) != 1 || start.RequiredCapabilities[0] != api.CapabilityTunMode {
		t.Fatalf("required capabilities = %+v", start.RequiredCapabilities)
	}

	if _, err := svc.EngineStop(ctx, api.EngineStopRequest{}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(privileged.runtimeStops) != 1 {
		t.Fatalf("runtime stops = %d, want provider-hosted stop", len(privileged.runtimeStops))
	}
}

func TestEngineStartUsesProviderHostedOwnerOnLinux(t *testing.T) {
	oldGOOS := runtimeGOOS
	runtimeGOOS = "linux"
	t.Cleanup(func() { runtimeGOOS = oldGOOS })

	privileged := readyFakePrivilegedProvider()
	privileged.prepare = map[string]api.PrepareFeatureReply{
		api.CapabilityTunMode: {Feature: api.CapabilityTunMode, State: api.CapabilityAvailable},
	}
	svc := newTestServiceWithPlatform(t, nil, privileged)
	ctx := context.Background()
	profileID := createProfileWithContent(t, svc, ctx, "linux-tun-target", `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: profileID}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(privileged.runtimeStarts) != 1 {
		t.Fatalf("runtime starts = %d, want provider-hosted start", len(privileged.runtimeStarts))
	}
	if start := privileged.runtimeStarts[0]; start.Mode != api.RuntimeModeMachineNetwork || start.ProfileID != profileID {
		t.Fatalf("runtime start = %+v", start)
	}
}

func TestEngineStartUsesNetworkExtensionOwnerOnDarwin(t *testing.T) {
	oldGOOS := runtimeGOOS
	runtimeGOOS = "darwin"
	t.Cleanup(func() { runtimeGOOS = oldGOOS })

	privileged := readyFakePrivilegedProvider()
	extension := &fakeNetworkExtensionRuntime{
		status: api.NetworkExtensionStatus{
			Installed:  true,
			Reachable:  true,
			Authorized: true,
			Capabilities: []api.Capability{
				{Name: api.CapabilityTunMode, State: api.CapabilityAvailable},
			},
		},
	}
	svc := newTestServiceWithPlatformAndExtension(t, nil, privileged, extension)
	ctx := context.Background()
	profileID := createProfileWithContent(t, svc, ctx, "darwin-tun-target", `{"inbounds":[{"type":"tun"}],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: profileID}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(extension.starts) != 1 {
		t.Fatalf("network extension starts = %d, want one", len(extension.starts))
	}
	start := extension.starts[0]
	if start.Mode != api.RuntimeModeAppleNetworkExtension || start.ProfileID != profileID {
		t.Fatalf("network extension start = %+v", start)
	}
	if len(privileged.runtimeStarts) != 0 || len(privileged.prepared) != 0 {
		t.Fatalf("privileged path used on Darwin: starts=%+v prepared=%+v", privileged.runtimeStarts, privileged.prepared)
	}

	if _, err := svc.EngineStop(ctx, api.EngineStopRequest{}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(extension.stops) != 1 {
		t.Fatalf("network extension stops = %d, want one", len(extension.stops))
	}
}

func createValidProfile(t *testing.T, svc *Service, ctx context.Context, name string) string {
	t.Helper()
	return createProfileWithContent(t, svc, ctx, name, `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)
}

func createProfileWithContent(t *testing.T, svc *Service, ctx context.Context, name, content string) string {
	t.Helper()
	createReply, err := svc.CreateProfile(ctx, api.CreateProfileRequest{
		Name:    name,
		Content: content,
	})
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return createReply.Profile.ID
}

func startEngineForProxyTest(t *testing.T, svc *Service) {
	t.Helper()
	ctx := context.Background()
	fake := &fakeRuntimeOwner{}
	svc.engine.runtimeOwnerFactory = func(RuntimeStartTarget) RuntimeOwner {
		return fake
	}
	profileID := createValidProfile(t, svc, ctx, "proxy-engine")
	if _, err := svc.ActivateProfile(ctx, api.ActivateProfileRequest{ProfileID: profileID}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if _, err := svc.EngineStart(ctx, api.EngineStartRequest{}); err != nil {
		t.Fatalf("engine start: %v", err)
	}
}
