package sovereignchain

import (
	"encoding/json"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/types"
)

type ChainState struct {
	mu      sync.RWMutex
	storage interfaces.Storage
	mempool []types.Transaction
}

func NewChainState(store interfaces.Storage) *ChainState {
	return &ChainState{
		storage: store,
		mempool: make([]types.Transaction, 0),
	}
}

func (cs *ChainState) Apply(blockData []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var block types.Block
	if err := json.Unmarshal(blockData, &block); err != nil {
		return nil, fmt.Errorf("invalid block data: %w", err)
	}

	key := fmt.Sprintf("block_%d", block.Index)
	if err := cs.storage.Put([]byte(key), blockData); err != nil {
		return nil, fmt.Errorf("failed to persist block: %w", err)
	}

	return []byte(block.Hash), nil
}

func (cs *ChainState) GetStateRoot() []byte {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return []byte("latest_state_hash")
}
