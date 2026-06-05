package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestHelloReplyJSONShape(t *testing.T) {
	reply := HelloReply{
		APIVersion:             APIVersion,
		MinSupportedAPIVersion: MinSupportedAPIVersion,
		SchemaRevision:         SchemaRevision,
		AppVersion:             AppVersion,
		QKBoxDVersion:          QKBoxDVersion,
		RuntimeCapabilities:    RuntimeCapabilityShell(),
		PlatformCapabilities:   PlatformCapabilityShell(),
	}

	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}

	required := []string{
		`"api_version"`,
		`"min_supported_api_version"`,
		`"schema_revision"`,
		`"runtime_capabilities"`,
		`"platform_capabilities"`,
		`"state":"unavailable"`,
	}
	for _, needle := range required {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestMethodRegistry(t *testing.T) {
	if _, ok := MethodRegistry[MethodHello]; !ok {
		t.Fatalf("missing %s in method registry", MethodHello)
	}
}

func TestStructuredErrorJSONShape(t *testing.T) {
	err := VersionUnsupported("0")
	got, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, needle := range []string{`"code":"IPC_VERSION_UNSUPPORTED"`, `"recoverable":false`, `"user_action"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestNewMethodRegistryEntries(t *testing.T) {
	newMethods := []string{
		MethodCreateProfile, MethodUpdateProfileDraft, MethodDeleteProfile,
		MethodListProfiles, MethodGetProfile,
		MethodAssetCreateProfileSubscription, MethodAssetListProfileSubscriptions,
		MethodAssetRefreshProfileSubscription, MethodAssetDeleteProfileSubscription,
		MethodAssetCreateDataAsset, MethodAssetListDataAssets,
		MethodAssetRefreshDataAsset, MethodAssetDeleteDataAsset,
		MethodValidateProfileDraft, MethodGetProfileDiagnostics,
		MethodCreateProfileSnapshot, MethodActivateProfileSnapshot,
		MethodGetActiveProfile, MethodGetActiveSnapshot,
		MethodListSnapshots, MethodRollbackToSnapshot,
	}
	for _, m := range newMethods {
		if _, ok := MethodRegistry[m]; !ok {
			t.Fatalf("missing %s in method registry", m)
		}
	}
}

func TestCreateProfileReplyJSONShape(t *testing.T) {
	reply := CreateProfileReply{}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"profile"`, `"id"`, `"name"`, `"created_at"`, `"updated_at"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestSnapshotJSONShape(t *testing.T) {
	reply := CreateProfileSnapshotReply{}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"snapshot"`, `"profile_id"`, `"validation_status"`, `"created_at"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestDiagnosticsJSONShape(t *testing.T) {
	reply := ValidateProfileDraftReply{}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"diagnostics"`, `"profile_id"`, `"status"`, `"entries"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestAssetPlaneJSONShape(t *testing.T) {
	values := []interface{}{
		CreateProfileSubscriptionReply{},
		ListProfileSubscriptionsReply{},
		RefreshProfileSubscriptionReply{},
		CreateDataAssetReply{},
		ListDataAssetsReply{},
		RefreshDataAssetReply{},
	}
	needles := []string{
		`"subscription"`,
		`"subscriptions"`,
		`"diagnostics"`,
		`"asset"`,
		`"assets"`,
		`"asset"`,
	}
	for i, value := range values {
		got, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(got), needles[i]) {
			t.Fatalf("expected %s in %s", needles[i], got)
		}
	}
}

func contains(value, needle string) bool {
	return strings.Contains(value, needle)
}

func TestEngineStatusJSONShape(t *testing.T) {
	status := EngineStatus{}
	got, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"state"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestEngineMethodRegistry(t *testing.T) {
	newMethods := []string{
		MethodEngineStart, MethodEngineStop, MethodEngineGetStatus,
		MethodEngineReload,
		MethodEngineSubscribeStatus, MethodEngineSubscribeLogs,
		MethodEngineSubscribeTraffic, MethodEngineSubscribeConnections,
		MethodEngineGetRuntimeCapabilities, MethodEngineListGroups,
		MethodEngineSelectOutbound, MethodEngineURLTest,
		MethodEngineCloseConnection, MethodEngineCloseAllConnections,
	}
	for _, m := range newMethods {
		if _, ok := MethodRegistry[m]; !ok {
			t.Fatalf("missing %s in method registry", m)
		}
	}
}

func TestRuntimeObservabilityJSONShape(t *testing.T) {
	values := []interface{}{
		RuntimeLogEntry{},
		TrafficSnapshot{},
		ConnectionSnapshot{},
		OutboundGroup{},
		URLTestResult{},
	}
	needles := []string{
		`"message"`,
		`"upload_total"`,
		`"connections"`,
		`"outbounds"`,
		`"outbound"`,
	}
	for i, value := range values {
		got, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !contains(string(got), needles[i]) {
			t.Fatalf("expected %s in %s", needles[i], got)
		}
	}
}

func TestPlatformMethodRegistry(t *testing.T) {
	newMethods := []string{
		MethodPlatformGetCapabilities,
		MethodPlatformGetPrivilegedProviderStatus,
		MethodPlatformPrepareFeature,
		MethodPlatformRunRepairAction,
		MethodPlatformGetSystemProxyStatus,
		MethodPlatformSetSystemProxyEnabled,
	}
	for _, m := range newMethods {
		if _, ok := MethodRegistry[m]; !ok {
			t.Fatalf("missing %s in method registry", m)
		}
	}
}

func TestSystemProxyStatusJSONShape(t *testing.T) {
	reply := GetSystemProxyStatusReply{
		Available:  true,
		Supported:  true,
		OSEnabled:  true,
		QKBoxOwned: true,
		Address:    "127.0.0.1",
		Port:       7890,
	}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"available"`, `"supported"`, `"os_enabled"`, `"qkbox_owned"`, `"address"`, `"port"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestEngineReloadJSONShape(t *testing.T) {
	reply := EngineReloadReply{
		Outcome:            ReloadOutcomeSuccess,
		TargetSnapshotID:   "target",
		PreviousSnapshotID: "previous",
		ActiveSnapshotID:   "target",
	}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"outcome"`, `"target_snapshot_id"`, `"previous_snapshot_id"`, `"active_snapshot_id"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestPrivilegedProviderStatusJSONShape(t *testing.T) {
	reply := GetPrivilegedProviderStatusReply{
		Status: PrivilegedProviderStatus{
			Installed:       true,
			Reachable:       true,
			Authenticated:   true,
			Version:         "0.1.0",
			ExpectedVersion: "0.1.0",
			Endpoint:        "provider",
			OwnerState:      &ProviderOwnerState{Owned: true, Stale: true, UID: "1000", SessionID: "session", RuntimeID: "runtime", SnapshotID: "snapshot", Mode: RuntimeModeMachineNetwork, RepairActions: []string{RepairActionClearMachineNetworkOwner}},
			Capabilities:    []Capability{{Name: CapabilityTunMode, State: CapabilityAvailable}},
		},
	}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"status"`, `"installed"`, `"reachable"`, `"authenticated"`, `"expected_version"`, `"owner_state"`, `"capabilities"`, `"stale"`, `"snapshot_id"`, `"repair_actions"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestNetworkExtensionStatusJSONShape(t *testing.T) {
	status := NetworkExtensionStatus{
		Installed:    true,
		Reachable:    true,
		Authorized:   true,
		BundleID:     "dev.qkbox.network-extension",
		Version:      "0.1.0",
		OwnerState:   &ProviderOwnerState{Owned: true, SessionID: "session", RuntimeID: "runtime", SnapshotID: "snapshot", Mode: RuntimeModeAppleNetworkExtension},
		Capabilities: []Capability{{Name: CapabilityTunMode, State: CapabilityAvailable}},
	}
	got, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"installed"`, `"reachable"`, `"authorized"`, `"bundle_id"`, `"owner_state"`, `"capabilities"`, `"apple_network_extension"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}

func TestPrepareFeatureJSONShape(t *testing.T) {
	reply := PrepareFeatureReply{Feature: CapabilityTunMode, State: CapabilityUnavailable, Reason: "Privileged network mutation is unavailable."}
	got, err := json.Marshal(reply)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"feature"`, `"state"`, `"reason"`} {
		if !contains(string(got), needle) {
			t.Fatalf("expected %s in %s", needle, got)
		}
	}
}
