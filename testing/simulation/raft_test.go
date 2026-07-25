package simulation

import (
	"context"
	"testing"
	"time"

	"sovereign-chain/modules/consensus"
)

func TestRaftConsensusProposal(t *testing.T) {
	raftNode := consensus.NewRaftConsensus("node_raft_1", true)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	data := []byte(`{"tx":"transfer_100_tokens"}`)
	if err := raftNode.Propose(ctx, data); err != nil {
		t.Fatalf("Raft proposal failed: %v", err)
	}

	select {
	case commit := <-raftNode.Commit():
		if string(commit) != string(data) {
			t.Fatalf("Committed data mismatch. Expected %s, got %s", string(data), string(commit))
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for Raft commit channel")
	}

	t.Logf("Raft consensus proposal and commit verified successfully")
}
