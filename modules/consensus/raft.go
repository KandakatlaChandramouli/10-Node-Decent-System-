package consensus

import (
	"context"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type RaftConsensus struct {
	mu         sync.RWMutex
	nodeID     string
	commitChan chan []byte
	isLeader   bool
}

func NewRaftConsensus(nodeID string, isLeader bool) *RaftConsensus {
	return &RaftConsensus{
		nodeID:     nodeID,
		commitChan: make(chan []byte, 1000),
		isLeader:   isLeader,
	}
}

func (r *RaftConsensus) Propose(ctx context.Context, data []byte) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.isLeader {
		return fmt.Errorf("node %s is not the Raft leader", r.nodeID)
	}

	select {
	case r.commitChan <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *RaftConsensus) Commit() <-chan []byte {
	return r.commitChan
}

func (r *RaftConsensus) SetLeader(isLeader bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isLeader = isLeader
}

var _ interfaces.Consensus = (*RaftConsensus)(nil)
