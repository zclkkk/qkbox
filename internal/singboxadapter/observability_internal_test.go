package singboxadapter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zclkkk/qkbox/shared/api"
)

type logSink struct {
	entries []string
}

func (s *logSink) PublishRuntimeLog(_, _, message string) {
	s.entries = append(s.entries, message)
}

func TestRuntimeLogWriterPublishesLog(t *testing.T) {
	sink := &logSink{}
	runtimeLogWriter{sink: sink}.WriteMessage(4, "hello")
	if len(sink.entries) != 1 || sink.entries[0] != "hello" {
		t.Fatalf("entries = %#v", sink.entries)
	}
}

func TestAdapterListsAndSelectsGroups(t *testing.T) {
	config := `{
		"inbounds": [],
		"outbounds": [
			{"type": "direct", "tag": "direct-a"},
			{"type": "direct", "tag": "direct-b"},
			{"type": "selector", "tag": "select", "outbounds": ["direct-a", "direct-b"], "default": "direct-a"}
		]
	}`
	adapter := NewAdapter()
	if err := adapter.Start(context.Background(), config); err != nil {
		skipIfClashAPINotIncluded(t, err)
		t.Fatalf("start: %v", err)
	}
	defer adapter.Stop()

	groups, structured := adapter.ListGroups()
	if structured != nil {
		t.Fatalf("list groups: %v", structured)
	}
	if len(groups) != 1 || groups[0].Tag != "select" || groups[0].Selected != "direct-a" {
		t.Fatalf("groups = %#v", groups)
	}

	group, structured := adapter.SelectOutbound("select", "direct-b")
	if structured != nil {
		t.Fatalf("select outbound: %v", structured)
	}
	if group.Selected != "direct-b" {
		t.Fatalf("selected = %s", group.Selected)
	}

	_, structured = adapter.SelectOutbound("missing", "direct-b")
	if structured == nil || structured.Code != api.ErrorRuntimeGroupNotFound {
		t.Fatalf("expected group not found, got %v", structured)
	}

	_, structured = adapter.URLTest(context.Background(), "select", time.Second)
	if structured == nil || structured.Code != api.ErrorObservabilityUnsupported {
		t.Fatalf("expected URLTest unsupported, got %v", structured)
	}
}

func skipIfClashAPINotIncluded(t *testing.T, err error) {
	t.Helper()
	if strings.Contains(err.Error(), "clash api is not included") {
		t.Skip("sing-box clash api is not included in this build")
	}
}
