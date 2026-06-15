package singboxadapter_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zclkkk/qkbox/internal/singboxadapter"
)

func TestAdapterStartCloseMinimal(t *testing.T) {
	config := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`

	adapter := singboxadapter.NewAdapter()
	ctx := context.Background()

	err := adapter.Start(ctx, config)
	if err != nil {
		skipIfClashAPINotIncluded(t, err)
		t.Fatalf("Failed to start minimal sing-box config: %v", err)
	}

	err = adapter.Stop()
	if err != nil {
		t.Fatalf("Failed to stop minimal sing-box config: %v", err)
	}
}

func skipIfClashAPINotIncluded(t *testing.T, err error) {
	t.Helper()
	if strings.Contains(err.Error(), "clash api is not included") {
		t.Skip("sing-box clash api is not included in this build")
	}
}
