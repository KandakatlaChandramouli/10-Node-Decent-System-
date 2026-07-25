package runtime

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"sovereign-chain/core/interfaces"
)

type ReplayEntry struct {
	Sequence  uint64    `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Payload   []byte    `json:"payload"`
	StateRoot string    `json:"state_root"`
}

type ReplayLog struct {
	mu      sync.RWMutex
	Entries []ReplayEntry `json:"entries"`
}

func NewReplayLog() *ReplayLog {
	return &ReplayLog{
		Entries: make([]ReplayEntry, 0),
	}
}

func (rl *ReplayLog) Record(payload []byte, stateRoot string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry := ReplayEntry{
		Sequence:  uint64(len(rl.Entries) + 1),
		Timestamp: time.Now().UTC(),
		Payload:   payload,
		StateRoot: stateRoot,
	}
	rl.Entries = append(rl.Entries, entry)
}

func (rl *ReplayLog) Serialize() ([]byte, error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return json.MarshalIndent(rl.Entries, "", "  ")
}

type ReplayEngine struct {
	stateMachine interfaces.StateMachine
}

func NewReplayEngine(sm interfaces.StateMachine) *ReplayEngine {
	return &ReplayEngine{stateMachine: sm}
}

func (re *ReplayEngine) Replay(logData []byte) (string, error) {
	var entries []ReplayEntry
	if err := json.Unmarshal(logData, &entries); err != nil {
		return "", fmt.Errorf("failed to parse replay log: %w", err)
	}

	var finalRoot string
	for _, entry := range entries {
		root, err := re.stateMachine.Apply(entry.Payload)
		if err != nil {
			return "", fmt.Errorf("replay failed at sequence %d: %w", entry.Sequence, err)
		}
		finalRoot = string(root)
		if entry.StateRoot != "" && finalRoot != entry.StateRoot {
			return "", fmt.Errorf("determinism violation at sequence %d! Expected root %s, got %s", entry.Sequence, entry.StateRoot, finalRoot)
		}
	}

	return finalRoot, nil
}
