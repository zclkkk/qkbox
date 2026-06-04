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
