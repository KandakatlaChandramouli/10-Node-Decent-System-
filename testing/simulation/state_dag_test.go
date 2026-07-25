package simulation

import (
	"os"
	"testing"

	sovereignchain "sovereign-chain/applications/sovereign-chain"
	"sovereign-chain/modules/storage"
)

func TestDecoupledStateMachineExecution(t *testing.T) {
	dir := "./test_data_dag_pebble"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Failed to init storage: %v", err)
	}
	defer store.Close()

	stateMachine := sovereignchain.NewChainState(store)

	mockCommitLog := []byte(`{"index":1,"hash":"0000a1b2c3d4e5f6","transactions":[]}`)

	root, err := stateMachine.Apply(mockCommitLog)
	if err != nil {
		t.Fatalf("State machine execution failed: %v", err)
	}

	if len(root) == 0 {
		t.Fatalf("Expected valid state root hash, got empty byte array")
	}

	t.Logf("State Machine executed commit. Root: %s", string(root))
}
