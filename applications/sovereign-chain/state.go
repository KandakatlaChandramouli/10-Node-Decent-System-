package sovereignchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/types"
)

type ChainState struct {
	mu        sync.RWMutex
	storage   interfaces.Storage
	stateRoot string
}

func NewChainState(store interfaces.Storage) *ChainState {
	return &ChainState{
		storage:   store,
		stateRoot: "genesis_root",
	}
}

func (cs *ChainState) Apply(commitLog []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var block types.Block
	if err := json.Unmarshal(commitLog, &block); err != nil {
		return nil, fmt.Errorf("state machine failed to unmarshal commit: %w", err)
	}

	key := fmt.Sprintf("state_block_%d", block.Index)
	if err := cs.storage.Put([]byte(key), commitLog); err != nil {
		return nil, fmt.Errorf("state persistence error: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(cs.stateRoot + block.Hash))
	cs.stateRoot = hex.EncodeToString(h.Sum(nil))

	return []byte(cs.stateRoot), nil
}

func (cs *ChainState) GetStateRoot() []byte {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return []byte(cs.stateRoot)
}

var _ interfaces.StateMachine = (*ChainState)(nil)
