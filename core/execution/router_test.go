package execution

import (
	"context"
	"crypto/sha256"
	"sovereign-chain/modules/privacy"
	"testing"
	"time"
)

// MockNetwork for Execution Router Tests
type MockNetwork struct {
	Payloads map[string][]byte
}

func (m *MockNetwork) RequestPayload(ctx context.Context, txID string, jurisdiction string) ([]byte, error) {
	if data, ok := m.Payloads[txID]; ok {
		return data, nil
	}
	return nil, context.DeadlineExceeded
}

func TestSovereignExecutionRouter(t *testing.T) {
	// 1. Initialize Subsystems
	vault := privacy.NewJurisdictionVault("AE-DU")
	net := &MockNetwork{Payloads: make(map[string][]byte)}
	fetcher := privacy.NewSideChannelFetcher(net, vault)
	telemetry := privacy.NewTelemetryPipeline(1)

	router := NewSovereignExecutionRouter(vault, fetcher, telemetry)

	// 2. Prepare Data
	data := []byte("smart-contract-execution-payload")
	hash := sha256.Sum256(data)
	anchor := privacy.AnchoredTransaction{
		TxID:             "tx-exe-1",
		JurisdictionCode: "AE-DU",
		PayloadHash:      hash,
	}

	// Seed network (simulating the payload arriving via P2P after the Raft anchor)
	net.Payloads["tx-exe-1"] = data

	// 3. Execute State Transition
	err := router.Apply(anchor)
	if err != nil {
		t.Fatalf("FAULT: Execution router failed to process state transition: %v", err)
	}

	// 4. Assert Global State (Raft Anchor)
	if router.GlobalState["tx-exe-1"] != hash {
		t.Fatal("FAULT: Global state anchor missing, consensus fracture detected")
	}

	// 5. Assert ZK Telemetry Generation
	time.Sleep(50 * time.Millisecond) // Allow async worker to process
	if _, ok := telemetry.Registry.Load("tx-exe-1"); !ok {
		t.Fatal("FAULT: Execution router failed to emit ZK telemetry trace")
	}
}
