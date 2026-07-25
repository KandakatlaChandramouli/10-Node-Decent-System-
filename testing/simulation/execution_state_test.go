package simulation

import (
	"os"
	"testing"

	sovereignchain "sovereign-chain/applications/sovereign-chain"
	"sovereign-chain/modules/storage"
)

func TestMerkleAuthenticatedState(t *testing.T) {
	dir := "./test_data_merkle"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Close()

	merkleState := sovereignchain.NewMerkleState(store)

	changes := map[string][]byte{
		"account_0x1": []byte("balance_100"),
		"account_0x2": []byte("balance_250"),
	}

	root, err := merkleState.ApplyChanges(changes)
	if err != nil {
		t.Fatalf("Failed to apply state changes: %v", err)
	}

	if len(root) != 32 {
		t.Fatalf("Expected 32-byte SHA-256 Merkle root, got %d bytes", len(root))
	}

	t.Logf("Merkle State Root calculated successfully: %x", root)
}
