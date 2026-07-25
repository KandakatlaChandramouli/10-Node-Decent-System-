package simulation

import (
	"context"
	"testing"
	"time"

	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/security"
	"sovereign-chain/modules/consensus"
	"sovereign-chain/modules/networking"
	"sovereign-chain/modules/storage"
	"sovereign-chain/sdk/plugin"
)

func TestKademliaDHTAndPeerScoring(t *testing.T) {
	dht := networking.NewKademliaDHT("node_alpha")

	peer := networking.PeerInfo{
		ID:        "node_beta",
		Address:   "192.168.1.50:9090",
		Score:     100,
		LastSeen:  time.Now(),
		LatencyMs: 12,
	}

	dht.AddPeer(peer)
	dht.AdjustPeerScore("node_beta", 25)

	if score := dht.GetPeerScore("node_beta"); score != 125 {
		t.Fatalf("Peer score mismatch. Expected 125, got %d", score)
	}

	t.Logf("Kademlia DHT & Peer Scoring verified successfully")
}

func TestDynamicRaftMembershipAndReadIndex(t *testing.T) {
	memStore := storage.NewMemoryStorage()
	peers := []string{"node1", "node2"}

	raftNode := consensus.NewRaftEngine("node1", peers, memStore)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = raftNode.Start(ctx)
	defer func() { _ = raftNode.Stop(ctx) }()

	// Simulate election grant to set node1 as leader for test
	raftNode.HandleRequestVote(consensus.RequestVoteArgs{
		Term:         1,
		CandidateID:  "node1",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})

	err := raftNode.ProposeMembershipChange(consensus.MembershipChange{
		Type:   consensus.AddNode,
		NodeID: "node3",
	})
	if err != nil {
		t.Fatalf("Failed to propose membership change: %v", err)
	}

	readResp := raftNode.GetReadIndex()
	if !readResp.Success {
		t.Fatalf("ReadIndex query failed on leader")
	}

	t.Logf("Dynamic Raft Membership & ReadIndex verified successfully")
}

func TestCapabilityAuthorizationAndAuditLog(t *testing.T) {
	auth := security.NewAuthorizationEngine()

	auth.GrantCapability("node_01", security.CapStateWrite)

	if !auth.Authorize("node_01", security.CapStateWrite, "commit_block") {
		t.Fatalf("Authorization check failed for granted capability")
	}

	if auth.Authorize("node_01", security.CapAdminAccess, "purge_db") {
		t.Fatalf("Authorization allowed ungranted capability")
	}

	logs := auth.GetAuditLogs()
	if len(logs) != 2 {
		t.Fatalf("Expected 2 audit log entries, got %d", len(logs))
	}

	t.Logf("Capability Authorization & Audit Logging verified successfully")
}

type MockPlugin struct{}

func (m *MockPlugin) Name() string    { return "mock_analytics_plugin" }
func (m *MockPlugin) Version() string { return "1.0.0" }
func (m *MockPlugin) Init(store interfaces.Storage) error {
	return nil
}

func TestSDKPluginFramework(t *testing.T) {
	pm := plugin.NewPluginManager()
	mockPlg := &MockPlugin{}

	if err := pm.RegisterPlugin(mockPlg); err != nil {
		t.Fatalf("Failed to register SDK plugin: %v", err)
	}

	p, found := pm.GetPlugin("mock_analytics_plugin")
	if !found || p.Version() != "1.0.0" {
		t.Fatalf("Plugin lookup failed")
	}

	t.Logf("Developer SDK Plugin Framework verified successfully")
}
