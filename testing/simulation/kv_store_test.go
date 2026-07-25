package simulation

import (
	"encoding/json"
	"os"
	"testing"

	kvstore "sovereign-chain/applications/kv-store"
	"sovereign-chain/modules/storage"
)

func TestDistributedKVStoreApplication(t *testing.T) {
	dir := "./test_data_kv"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Close()

	kv := kvstore.NewKVState(store)

	// Apply PUT operation
	putPayload, _ := json.Marshal(kvstore.KVPayload{
		Op:    kvstore.OpPut,
		Key:   "cluster_config",
		Value: []byte("active_nodes=100"),
	})

	root1, err := kv.Apply(putPayload)
	if err != nil {
		t.Fatalf("Failed to apply KV PUT payload: %v", err)
	}

	val, found := kv.Get("cluster_config")
	if !found || string(val) != "active_nodes=100" {
		t.Fatalf("KV store state mismatch. Expected active_nodes=100, got %s", string(val))
	}

	// Apply DELETE operation
	delPayload, _ := json.Marshal(kvstore.KVPayload{
		Op:  kvstore.OpDelete,
		Key: "cluster_config",
	})

	root2, err := kv.Apply(delPayload)
	if err != nil {
		t.Fatalf("Failed to apply KV DELETE payload: %v", err)
	}

	if string(root1) == string(root2) {
		t.Fatalf("Expected state root to update after deletion")
	}

	_, foundAfterDelete := kv.Get("cluster_config")
	if foundAfterDelete {
		t.Fatalf("Expected key 'cluster_config' to be deleted")
	}

	t.Logf("Distributed KV Store executed successfully. Final Root: %s", string(root2))
}
