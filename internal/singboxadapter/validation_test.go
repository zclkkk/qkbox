package singboxadapter_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/zclkkk/qkbox/internal/singboxadapter"
	"github.com/zclkkk/qkbox/shared/model"
)

func TestValidateAcceptsMinimalConfig(t *testing.T) {
	diag := singboxadapter.Validate(`{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`)
	if diag.Status != model.ValidationStatusValid {
		t.Fatalf("diagnostics = %+v", diag)
	}
}

func TestValidateRejectsInvalidJSON(t *testing.T) {
	diag := singboxadapter.Validate(`not json`)
	if diag.Status != model.ValidationStatusInvalid {
		t.Fatalf("status = %s, want invalid", diag.Status)
	}
	if len(diag.Entries) != 1 || diag.Entries[0].Severity != model.SeverityError || diag.Entries[0].Message == "" {
		t.Fatalf("entries = %+v", diag.Entries)
	}
}

func TestValidateRejectsSingboxSemanticError(t *testing.T) {
	diag := singboxadapter.Validate(`{"inbounds":[],"outbounds":[{"type":"missing-protocol","tag":"bad"}]}`)
	if diag.Status != model.ValidationStatusInvalid {
		t.Fatalf("status = %s, want invalid", diag.Status)
	}
}

func TestValidateDoesNotStartListeners(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	config := fmt.Sprintf(`{"inbounds":[{"type":"mixed","tag":"mixed-in","listen":"127.0.0.1","listen_port":%d}],"outbounds":[{"type":"direct","tag":"direct"}]}`, port)
	diag := singboxadapter.Validate(config)
	if diag.Status != model.ValidationStatusValid {
		t.Fatalf("diagnostics = %+v", diag)
	}
}
