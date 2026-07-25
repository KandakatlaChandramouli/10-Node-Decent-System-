package kvstore

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"sovereign-chain/core/interfaces"
)

type OperationType string

const (
	OpPut    OperationType = "PUT"
	OpDelete OperationType = "DELETE"
)

type KVPayload struct {
	Op    OperationType `json:"op"`
	Key   string        `json:"key"`
	Value []byte        `json:"value,omitempty"`
}

type KVState struct {
	mu        sync.RWMutex
	storage   interfaces.Storage
	inMemory  map[string][]byte
	stateRoot string
}

func NewKVState(store interfaces.Storage) *KVState {
	return &KVState{
		storage:   store,
		inMemory:  make(map[string][]byte),
		stateRoot: "kv_genesis_root",
	}
}

func (kv *KVState) Apply(commitLog []byte) ([]byte, error) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	var payload KVPayload
	if err := json.Unmarshal(commitLog, &payload); err != nil {
		return nil, fmt.Errorf("kv-store failed to unmarshal commit payload: %w", err)
	}

	switch payload.Op {
	case OpPut:
		kv.inMemory[payload.Key] = payload.Value
		if err := kv.storage.Put([]byte(payload.Key), payload.Value); err != nil {
			return nil, fmt.Errorf("kv-store storage put error: %w", err)
		}
	case OpDelete:
		delete(kv.inMemory, payload.Key)
		if err := kv.storage.Delete([]byte(payload.Key)); err != nil {
			return nil, fmt.Errorf("kv-store storage delete error: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported kv operation: %s", payload.Op)
	}

	kv.recalculateRoot()
	return []byte(kv.stateRoot), nil
}

func (kv *KVState) recalculateRoot() {
	keys := make([]string, 0, len(kv.inMemory))
	for k := range kv.inMemory {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write(kv.inMemory[k])
	}
	kv.stateRoot = fmt.Sprintf("%x", h.Sum(nil))
}

func (kv *KVState) GetStateRoot() []byte {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	return []byte(kv.stateRoot)
}

func (kv *KVState) Get(key string) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	val, ok := kv.inMemory[key]
	return val, ok
}

var _ interfaces.StateMachine = (*KVState)(nil)
