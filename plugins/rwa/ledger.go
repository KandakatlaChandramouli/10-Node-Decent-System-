package rwa

import (
	"encoding/json"
	"errors"
	"sync"
)

// RWAStateTransition defines the payload structure for asset tokenization
type RWAStateTransition struct {
	AssetID      string `json:"asset_id"`
	TotalShares  uint64 `json:"total_shares"`
	IssuerPubKey string `json:"issuer_pub_key"`
}

// Ledger acts as an isolated state machine for Real-World Assets
type Ledger struct {
	mu     sync.RWMutex
	Assets map[string]uint64 // AssetID -> TotalShares
}

func NewLedger() *Ledger {
	return &Ledger{
		Assets: make(map[string]uint64),
	}
}

// Execute complies with a generic StateMachine interface expected by the core router
func (l *Ledger) Execute(payload []byte) error {
	var transition RWAStateTransition
	if err := json.Unmarshal(payload, &transition); err != nil {
		return errors.New("RWA_FAULT: Invalid payload serialization")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.Assets[transition.AssetID]; exists {
		return errors.New("RWA_FAULT: Asset ID already registered in sovereign ledger")
	}

	// Fractionalize the asset and assign to issuer
	l.Assets[transition.AssetID] = transition.TotalShares
	return nil
}
