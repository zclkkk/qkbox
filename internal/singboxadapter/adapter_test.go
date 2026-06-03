package singboxadapter_test

import (
	"context"
	"testing"

	"github.com/zclkkk/qkbox/internal/singboxadapter"
)

func TestAdapterStartCloseMinimal(t *testing.T) {
	config := `{"inbounds":[],"outbounds":[{"type":"direct","tag":"direct"}]}`

	adapter := singboxadapter.NewAdapter()
	ctx := context.Background()

	err := adapter.Start(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start minimal sing-box config: %v", err)
	}

	err = adapter.Stop()
	if err != nil {
		t.Fatalf("Failed to stop minimal sing-box config: %v", err)
	}
}
