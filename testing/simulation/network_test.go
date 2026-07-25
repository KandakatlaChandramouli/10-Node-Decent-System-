package simulation

import (
	"context"
	"net/http"
	"testing"
	"time"

	"sovereign-chain/modules/networking"
)

func TestP2PNodeMetricsEndpoint(t *testing.T) {
	node := networking.NewP2PNode(9090, []string{"127.0.0.1:9091"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := node.Start(ctx); err != nil {
		t.Fatalf("Failed to start P2P node: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:9090/metrics")
	if err != nil {
		t.Fatalf("Failed to query metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 OK, got %d", resp.StatusCode)
	}
}
