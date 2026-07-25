package consensus

import (
	"fmt"
)

type MembershipChangeType string

const (
	AddNode    MembershipChangeType = "ADD_NODE"
	RemoveNode MembershipChangeType = "REMOVE_NODE"
)

type MembershipChange struct {
	Type   MembershipChangeType `json:"type"`
	NodeID string               `json:"node_id"`
}

type ReadIndexResponse struct {
	ReadIndex int  `json:"read_index"`
	Term      int  `json:"term"`
	Success   bool `json:"success"`
}

func (re *RaftEngine) ProposeMembershipChange(change MembershipChange) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if re.role != RoleLeader {
		return fmt.Errorf("only leader can propose membership changes")
	}

	switch change.Type {
	case AddNode:
		for _, peer := range re.peers {
			if peer == change.NodeID {
				return fmt.Errorf("node %s is already in dynamic membership list", change.NodeID)
			}
		}
		re.peers = append(re.peers, change.NodeID)
	case RemoveNode:
		updatedPeers := make([]string, 0, len(re.peers))
		found := false
		for _, peer := range re.peers {
			if peer != change.NodeID {
				updatedPeers = append(updatedPeers, peer)
			} else {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("node %s not found in dynamic membership list", change.NodeID)
		}
		re.peers = updatedPeers
	}

	re.persistState()
	return nil
}

func (re *RaftEngine) GetReadIndex() ReadIndexResponse {
	re.mu.RLock()
	defer re.mu.RUnlock()

	if re.role != RoleLeader {
		return ReadIndexResponse{Success: false}
	}

	return ReadIndexResponse{
		ReadIndex: re.commitIndex,
		Term:      re.currentTerm,
		Success:   true,
	}
}

func (re *RaftEngine) SetRole(role NodeRole) {
	re.mu.Lock()
	defer re.mu.Unlock()
	re.role = role
}
