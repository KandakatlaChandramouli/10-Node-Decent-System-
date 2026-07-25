package simulation

import (
	"context"
	"testing"
	"time"

	"sovereign-chain/modules/consensus"
	"sovereign-chain/modules/storage"
)

func TestRaftEngineReplicatedLogAndElection(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	peers := []string{"node1", "node2", "node3"}

	node1 := consensus.NewRaftEngine("node1", peers, memStore)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := node1.Start(ctx); err != nil {
		t.Fatalf("Failed to start Raft node: %v", err)
	}
	defer func() { _ = node1.Stop(ctx) }()

	time.Sleep(200 * time.Millisecond)

	// Simulate election victory
	voteArgs := consensus.RequestVoteArgs{
		Term:         1,
		CandidateID:  "node1",
		LastLogIndex: 0,
		LastLogTerm:  0,
	}
	reply := node1.HandleRequestVote(voteArgs)
	if !reply.VoteGranted {
		t.Fatalf("Expected vote to be granted for higher term")
	}

	// Test AppendEntries replication
	appendArgs := consensus.AppendEntriesArgs{
		Term:         1,
		LeaderID:     "node2",
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries: []consensus.LogEntry{
			{Index: 1, Term: 1, Data: []byte("replicated_tx_01")},
		},
		LeaderCommit: 1,
	}

	appReply := node1.HandleAppendEntries(appendArgs)
	if !appReply.Success {
		t.Fatalf("AppendEntries replication failed")
	}

	select {
	case commitData := <-node1.Commit():
		if string(commitData) != "replicated_tx_01" {
			t.Fatalf("Committed data mismatch. Expected 'replicated_tx_01', got '%s'", string(commitData))
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timed out waiting for Raft replicated log commit")
	}

	t.Logf("Raft Replicated Log Engine verified successfully")
}
