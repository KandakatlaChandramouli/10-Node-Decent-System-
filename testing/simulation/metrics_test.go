package simulation

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"sovereign-chain/modules/networking"
)

func TestPrometheusMetricsFormat(t *testing.T) {
	node := networking.NewP2PNode(9095, []string{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := node.Start(ctx); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:9095/metrics")
	if err != nil {
		t.Fatalf("Failed to query metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	if !strings.Contains(content, "sovereign_blocks_mined_total") {
		t.Fatalf("Expected metric sovereign_blocks_mined_total in response")
	}
}
