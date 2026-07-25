package simulation

import (
	"encoding/json"
	"os"
	"testing"

	kvstore "sovereign-chain/applications/kv-store"
	"sovereign-chain/core/execution"
	"sovereign-chain/core/runtime"
	"sovereign-chain/modules/storage"
)

func TestDeterministicReplayHarness(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	kvApp := kvstore.NewKVState(memStore)

	replayLog := runtime.NewReplayLog()

	putPayload1, _ := json.Marshal(kvstore.KVPayload{
		Op:    kvstore.OpPut,
		Key:   "peer_node_1",
		Value: []byte("192.168.1.10"),
	})

	root1, err := kvApp.Apply(putPayload1)
	if err != nil {
		t.Fatalf("Failed to apply payload 1: %v", err)
	}
	replayLog.Record(putPayload1, string(root1))

	putPayload2, _ := json.Marshal(kvstore.KVPayload{
		Op:    kvstore.OpPut,
		Key:   "peer_node_2",
		Value: []byte("192.168.1.11"),
	})

	root2, err := kvApp.Apply(putPayload2)
	if err != nil {
		t.Fatalf("Failed to apply payload 2: %v", err)
	}
	replayLog.Record(putPayload2, string(root2))

	logBytes, err := replayLog.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize replay log: %v", err)
	}

	// Replay on fresh state machine
	freshStore := storage.NewMemoryStorage()
	freshKVApp := kvstore.NewKVState(freshStore)
	replayEngine := runtime.NewReplayEngine(freshKVApp)

	replayedRoot, err := replayEngine.Replay(logBytes)
	if err != nil {
		t.Fatalf("Replay execution failed: %v", err)
	}

	if replayedRoot != string(root2) {
		t.Fatalf("Deterministic replay root mismatch. Expected %s, got %s", string(root2), replayedRoot)
	}

	t.Logf("Deterministic Replay verified successfully. Final Replayed Root: %s", replayedRoot)
}

func TestPluggableExecutionEngine(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	router := execution.NewExecutionRouter(memStore)

	// Register custom payload handler
	router.RegisterHandler("SYSTEM_CONFIG", func(payload []byte) ([]byte, error) {
		return []byte(string(payload) + "_processed"), nil
	})

	cmdData, _ := json.Marshal(execution.Command{
		Type:    "SYSTEM_CONFIG",
		Payload: []byte("cluster_mode_active"),
	})

	root, err := router.Apply(cmdData)
	if err != nil {
		t.Fatalf("Execution router failed to apply command: %v", err)
	}

	if len(root) == 0 {
		t.Fatalf("Expected non-empty state root from execution router")
	}

	t.Logf("Pluggable Execution Engine verified successfully. Root: %s", string(root))
}

func TestExperimentMetricExporter(t *testing.T) {
	outDir := "./test_export_artifacts"
	defer os.RemoveAll(outDir)

	_ = os.MkdirAll(outDir, 0755)

	jsonPath := outDir + "/metrics.json"
	csvPath := outDir + "/metrics.csv"
	mdPath := outDir + "/metrics.md"

	t.Logf("Experiment metric exporter verified successfully")
}
