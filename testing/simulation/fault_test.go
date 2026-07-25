package simulation

import (
	"context"
	"net/http"
	"testing"
	"time"

	"sovereign-chain/modules/networking"
)

func TestNodeShutdownAndRecovery(t *testing.T) {
	node := networking.NewP2PNode(9099, []string{})
	ctx, cancel := context.WithCancel(context.Background())

	if err := node.Start(ctx); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify server running
	resp, err := http.Get("http://127.0.0.1:9099/ping")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP 200 on ping, got error: %v", err)
	}
	resp.Body.Close()

	// Simulate sudden node crash / cancellation
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Verify server down
	_, err = http.Get("http://127.0.0.1:9099/ping")
	if err == nil {
		t.Fatalf("Expected connection refusal after node cancellation, but server responded")
	}
}
