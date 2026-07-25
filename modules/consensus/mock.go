package consensus

import (
	"context"

	"sovereign-chain/core/interfaces"
)

type MockConsensus struct {
	commitChan chan []byte
}

func NewMockConsensus() *MockConsensus {
	return &MockConsensus{
		commitChan: make(chan []byte, 100),
	}
}

func (m *MockConsensus) Propose(ctx context.Context, data []byte) error {
	m.commitChan <- data
	return nil
}

func (m *MockConsensus) Commit() <-chan []byte {
	return m.commitChan
}

var _ interfaces.Consensus = (*MockConsensus)(nil)
