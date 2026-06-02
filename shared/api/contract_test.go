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

func contains(value, needle string) bool {
	return strings.Contains(value, needle)
}
