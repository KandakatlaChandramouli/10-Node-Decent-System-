package simulation

import (
	"os"
	"testing"

	"sovereign-chain/modules/storage"
)

func TestPebbleStorageEngine(t *testing.T) {
	dir := "./test_data_pebble"
	defer os.RemoveAll(dir)

	store, err := storage.NewPebbleStorage(dir)
	if err != nil {
		t.Fatalf("Failed to initialize Pebble storage: %v", err)
	}
	defer store.Close()

	key := []byte("latest_block_hash")
	val := []byte("0000a1b2c3d4e5f6")

	if err := store.Put(key, val); err != nil {
		t.Fatalf("Failed to put key-value: %v", err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Failed to get value for key: %v", err)
	}

	if string(got) != string(val) {
		t.Fatalf("Expected %s, got %s", val, got)
	}
}
