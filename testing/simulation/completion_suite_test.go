package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"sovereign-chain/core/execution"
	"sovereign-chain/core/runtime"
	"sovereign-chain/core/security"
	"sovereign-chain/modules/consensus"
	"sovereign-chain/modules/storage"
)

func TestRaftLogCompactionAndSnapshotting(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	peers := []string{"node1", "node2"}

	raftNode := consensus.NewRaftEngine("node1", peers, memStore)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = raftNode.Start(ctx)
	defer func() { _ = raftNode.Stop(ctx) }()

	appendArgs := consensus.AppendEntriesArgs{
		Term:         1,
		LeaderID:     "node2",
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []consensus.LogEntry{
			{Index: 1, Term: 1, Data: []byte("entry_1")},
			{Index: 2, Term: 1, Data: []byte("entry_2")},
			{Index: 3, Term: 1, Data: []byte("entry_3")},
		},
		LeaderCommit: 3,
	}

	reply := raftNode.HandleAppendEntries(appendArgs)
	if !reply.Success {
		t.Fatalf("Failed to append entries")
	}

	err := raftNode.CompactLog(2, []byte("state_snapshot_at_index_2"))
	if err != nil {
		t.Fatalf("Log compaction failed: %v", err)
	}

	t.Logf("Raft Log Compaction & Snapshotting verified successfully")
}

func TestTransactionalExecutionAndRollback(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	router := execution.NewExecutionRouter(memStore)

	router.RegisterHandler("BANK_TRANSFER", func(payload []byte) ([]byte, error) {
		var req map[string]int
		_ = json.Unmarshal(payload, &req)
		if req["amount"] > 1000 {
			return nil, fmt.Errorf("amount exceeds transfer limit of 1000")
		}
		return []byte("transfer_success"), nil
	})

	validTx, _ := json.Marshal(execution.Command{
		Type:    "BANK_TRANSFER",
		Payload: []byte(`{"amount": 500}`),
	})

	receipt, err := router.ApplyTransactional("tx_101", validTx)
	if err != nil || receipt.Status != "COMMITTED" {
		t.Fatalf("Transaction commit failed: %v", err)
	}

	invalidTx, _ := json.Marshal(execution.Command{
		Type:    "BANK_TRANSFER",
		Payload: []byte(`{"amount": 5000}`),
	})

	failedReceipt, err := router.ApplyTransactional("tx_102", invalidTx)
	if err == nil || failedReceipt.Status != "ROLLED_BACK" {
		t.Fatalf("Expected transaction rollback on error")
	}

	t.Logf("Transactional Execution & Rollback verified successfully")
}

func TestNodeIdentityAndSecurity(t *testing.T) {
	identity, err := security.GenerateNodeIdentity()
	if err != nil {
		t.Fatalf("Node identity generation failed: %v", err)
	}

	msg := []byte("sovereign_runtime_payload")
	sig := identity.SignMessage(msg)

	if !security.VerifyMessageSignature(identity.PublicKey, msg, sig) {
		t.Fatalf("Signature verification failed")
	}

	t.Logf("Node Security & Ed25519 Identity verified successfully. Node ID: %s", identity.NodeID)
}

func TestRuntimeObservability(t *testing.T) {
	obs := runtime.NewRuntimeObservability()
	obs.RecordExecution(15 * time.Millisecond)
	obs.RecordExecution(25 * time.Millisecond)

	count, avgLat := obs.GetMetrics()
	if count != 2 || avgLat != 20 {
		t.Fatalf("Observability metrics mismatch. Expected count=2, avgLat=20, got count=%d, avgLat=%d", count, avgLat)
	}

	t.Logf("Runtime Observability & Latency Metrics verified successfully")
}
