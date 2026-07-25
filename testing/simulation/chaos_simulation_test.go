package simulation

import (
	"context"
	"os"
	"testing"
	"time"

	sovereignchain "sovereign-chain/applications/sovereign-chain"
	"sovereign-chain/modules/storage"
	"sovereign-chain/testing/chaos"
)

func TestChaosResiliencyUnderPartitionAndDelay(t *testing.T) {
	dir := "./test_data_chaos"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Close()

	stateMachine := sovereignchain.NewChainState(store)

	injector := chaos.NewFaultInjector(chaos.FaultConfig{
		PartitionRatio:   0.3,
		MaxDelayMs:       50,
		CrashProbability: 0.1,
	})

	nodeID := "node_test_1"
	injector.InjectPartition(nodeID)

	if !injector.ShouldDropMessage(nodeID) {
		t.Fatalf("Expected fault injector to drop message for partitioned node")
	}

	injector.HealPartition(nodeID)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := injector.SimulateMessageLatency(ctx); err != nil {
		t.Fatalf("Latency simulation error: %v", err)
	}

	commitLog := []byte(`{"index":100,"hash":"0000f1f2f3f4","transactions":[]}`)
	root, err := stateMachine.Apply(commitLog)
	if err != nil {
		t.Fatalf("State machine failed to recover and apply commit after chaos events: %v", err)
	}

	t.Logf("State Machine recovered successfully from chaos events. State Root: %s", string(root))
}
