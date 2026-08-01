package privacy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"
	"time"
)

// MockNetwork simulates the LibP2P routing layer for side-channel fetches
type MockNetwork struct {
	Payloads map[string][]byte
}

func (m *MockNetwork) RequestPayload(ctx context.Context, txID string, jurisdiction string) ([]byte, error) {
	if data, ok := m.Payloads[txID]; ok {
		return data, nil
	}
	return nil, context.DeadlineExceeded
}

func TestJurisdictionVault(t *testing.T) {
	vault := NewJurisdictionVault("AE-DU") // Node represents Dubai Jurisdiction
	data := []byte("classified-asset-deed")
	hash := sha256.Sum256(data)

	// TEST 1: Reject Foreign Data
	foreignPayload := PrivatePayload{TxID: "tx1", JurisdictionCode: "US-NY", EncryptedData: data}
	err := vault.StoreIfAuthorized(foreignPayload)
	if err == nil {
		t.Fatal("SECURITY FAULT: Vault accepted data from unauthorized foreign jurisdiction")
	}

	// TEST 2: Accept Authorized Data
	localPayload := PrivatePayload{TxID: "tx2", JurisdictionCode: "AE-DU", EncryptedData: data}
	err = vault.StoreIfAuthorized(localPayload)
	if err != nil {
		t.Fatalf("FAULT: Vault rejected authorized local data: %v", err)
	}

	// TEST 3: Cryptographic Verification on Retrieval
	retrieved, err := vault.RetrieveAndVerify("tx2", hash)
	if err != nil || !bytes.Equal(retrieved, data) {
		t.Fatalf("FAULT: Failed to retrieve or verify payload signature: %v", err)
	}
}

func TestSideChannelFetcher(t *testing.T) {
	vault := NewJurisdictionVault("AE-DU")
	net := &MockNetwork{Payloads: make(map[string][]byte)}
	fetcher := NewSideChannelFetcher(net, vault)

	data := []byte("p2p-fetched-payload")
	hash := sha256.Sum256(data)
	net.Payloads["tx-async-1"] = data // Seed the mock P2P network

	// Trigger async fetch and verify it stores correctly in the vault
	err := fetcher.FetchAndStore("tx-async-1", hash)
	if err != nil {
		t.Fatalf("FAULT: Side-channel fetcher failed to retrieve and store: %v", err)
	}
}

func TestZKTelemetryPipeline(t *testing.T) {
	pipeline := NewTelemetryPipeline(2)

	trace := ExecutionTrace{
		TxID:             "tx-zk-1",
		JurisdictionCode: "AE-DU",
	}

	// Emit trace non-blockingly
	pipeline.TraceQueue <- trace

	// Wait for async workers to process the cryptographic proof
	time.Sleep(100 * time.Millisecond)

	_, ok := pipeline.Registry.Load("tx-zk-1")
	if !ok {
		t.Fatal("FAULT: ZK proof was not generated and registered by the async pipeline")
	}
}
