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
