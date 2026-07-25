package simulation

import (
	"os"
	"testing"

	sovereignchain "sovereign-chain/applications/sovereign-chain"
	"sovereign-chain/modules/storage"
)

func TestMerkleInclusionProofVerification(t *testing.T) {
	dir := "./test_data_proof"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Close()

	merkleState := sovereignchain.NewMerkleState(store)

	changes := map[string][]byte{
		"balance_user1": []byte("1000_tokens"),
	}

	root, err := merkleState.ApplyChanges(changes)
	if err != nil {
		t.Fatalf("Failed to apply state changes: %v", err)
	}

	proof, val, err := merkleState.GetProof([]byte("balance_user1"))
	if err != nil {
		t.Fatalf("Failed to get Merkle proof: %v", err)
	}

	isValid := merkleState.VerifyProof([]byte("balance_user1"), val, proof, root)
	if !isValid {
		t.Fatalf("Merkle inclusion proof verification failed")
	}

	t.Logf("Merkle inclusion proof verified successfully against State Root: %x", root)
}
