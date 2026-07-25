package distlock

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"sovereign-chain/core/interfaces"
)

type LockAction string

const (
	ActionAcquire LockAction = "ACQUIRE"
	ActionRelease LockAction = "RELEASE"
)

type LockPayload struct {
	Action   LockAction `json:"action"`
	Resource string     `json:"resource"`
	Holder   string     `json:"holder"`
	TTLMs    int64      `json:"ttl_ms"`
}

type LockInfo struct {
	Holder    string    `json:"holder"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LockState struct {
	mu        sync.RWMutex
	storage   interfaces.Storage
	locks     map[string]LockInfo
	stateRoot string
}

func NewLockState(store interfaces.Storage) *LockState {
	return &LockState{
		storage:   store,
		locks:     make(map[string]LockInfo),
		stateRoot: "lock_genesis_root",
	}
}

func (ls *LockState) Apply(commitLog []byte) ([]byte, error) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	var payload LockPayload
	if err := json.Unmarshal(commitLog, &payload); err != nil {
		return nil, fmt.Errorf("dist-lock failed to unmarshal payload: %w", err)
	}

	now := time.Now()

	switch payload.Action {
	case ActionAcquire:
		if current, exists := ls.locks[payload.Resource]; exists {
			if now.Before(current.ExpiresAt) && current.Holder != payload.Holder {
				return nil, fmt.Errorf("resource %s already locked by %s", payload.Resource, current.Holder)
			}
		}
		ls.locks[payload.Resource] = LockInfo{
			Holder:    payload.Holder,
			ExpiresAt: now.Add(time.Duration(payload.TTLMs) * time.Millisecond),
		}

	case ActionRelease:
		current, exists := ls.locks[payload.Resource]
		if !exists {
			return nil, fmt.Errorf("resource %s is not locked", payload.Resource)
		}
		if current.Holder != payload.Holder {
			return nil, fmt.Errorf("unauthorized release attempt by %s for resource locked by %s", payload.Holder, current.Holder)
		}
		delete(ls.locks, payload.Resource)

	default:
		return nil, fmt.Errorf("unsupported lock action: %s", payload.Action)
	}

	ls.recalculateRoot()
	return []byte(ls.stateRoot), nil
}

func (ls *LockState) recalculateRoot() {
	h := sha256.New()
	for res, info := range ls.locks {
		h.Write([]byte(res))
		h.Write([]byte(info.Holder))
	}
	ls.stateRoot = fmt.Sprintf("%x", h.Sum(nil))
}

func (ls *LockState) GetStateRoot() []byte {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return []byte(ls.stateRoot)
}

func (ls *LockState) IsLocked(resource string) bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	info, exists := ls.locks[resource]
	if !exists {
		return false
	}
	return time.Now().Before(info.ExpiresAt)
}

var _ interfaces.StateMachine = (*LockState)(nil)
