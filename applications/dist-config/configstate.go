package distconfig

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"sovereign-chain/core/interfaces"
)

type ConfigAction string

const (
	ActionSetConfig ConfigAction = "SET"
	ActionDelConfig ConfigAction = "DEL"
)

type ConfigPayload struct {
	Action ConfigAction `json:"action"`
	Namespace string     `json:"namespace"`
	Key       string     `json:"key"`
	Value     string     `json:"value,omitempty"`
}

type ConfigState struct {
	mu        sync.RWMutex
	storage   interfaces.Storage
	configs   map[string]map[string]string
	stateRoot string
}

func NewConfigState(store interfaces.Storage) *ConfigState {
	return &ConfigState{
		storage:   store,
		configs:   make(map[string]map[string]string),
		stateRoot: "config_genesis_root",
	}
}

func (cs *ConfigState) Apply(commitLog []byte) ([]byte, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var payload ConfigPayload
	if err := json.Unmarshal(commitLog, &payload); err != nil {
		return nil, fmt.Errorf("dist-config failed to unmarshal payload: %w", err)
	}

	if _, exists := cs.configs[payload.Namespace]; !exists {
		cs.configs[payload.Namespace] = make(map[string]string)
	}

	switch payload.Action {
	case ActionSetConfig:
		cs.configs[payload.Namespace][payload.Key] = payload.Value
	case ActionDelConfig:
		delete(cs.configs[payload.Namespace], payload.Key)
	default:
		return nil, fmt.Errorf("unsupported config action: %s", payload.Action)
	}

	cs.recalculateRoot()
	return []byte(cs.stateRoot), nil
}

func (cs *ConfigState) recalculateRoot() {
	nsKeys := make([]string, 0, len(cs.configs))
	for ns := range cs.configs {
		nsKeys = append(nsKeys, ns)
	}
	sort.Strings(nsKeys)

	h := sha256.New()
	for _, ns := range nsKeys {
		h.Write([]byte(ns))
		keys := make([]string, 0, len(cs.configs[ns]))
		for k := range cs.configs[ns] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.Write([]byte(k))
			h.Write([]byte(cs.configs[ns][k]))
		}
	}
	cs.stateRoot = fmt.Sprintf("%x", h.Sum(nil))
}

func (cs *ConfigState) GetStateRoot() []byte {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return []byte(cs.stateRoot)
}

func (cs *ConfigState) GetConfig(ns, key string) (string, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if sub, exists := cs.configs[ns]; exists {
		val, ok := sub[key]
		return val, ok
	}
	return "", false
}

var _ interfaces.StateMachine = (*ConfigState)(nil)
