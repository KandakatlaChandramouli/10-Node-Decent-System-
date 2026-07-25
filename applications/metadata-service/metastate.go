package metadataservice

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"sovereign-chain/core/interfaces"
)

type MetaAction string

const (
	ActionPutMeta MetaAction = "PUT_META"
	ActionDelMeta MetaAction = "DEL_META"
)

type MetadataPayload struct {
	Action   MetaAction        `json:"action"`
	ObjectID string            `json:"object_id"`
	Metadata map[string]string `json:"metadata"`
}

type MetadataState struct {
	mu        sync.RWMutex
	storage   interfaces.Storage
	objects   map[string]map[string]string
	stateRoot string
}

func NewMetadataState(store interfaces.Storage) *MetadataState {
	return &MetadataState{
		storage:   store,
		objects:   make(map[string]map[string]string),
		stateRoot: "meta_genesis_root",
	}
}

func (ms *MetadataState) Apply(commitLog []byte) ([]byte, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var payload MetadataPayload
	if err := json.Unmarshal(commitLog, &payload); err != nil {
		return nil, fmt.Errorf("metadata-service failed to unmarshal payload: %w", err)
	}

	switch payload.Action {
	case ActionPutMeta:
		ms.objects[payload.ObjectID] = payload.Metadata
	case ActionDelMeta:
		delete(ms.objects, payload.ObjectID)
	default:
		return nil, fmt.Errorf("unsupported meta action: %s", payload.Action)
	}

	ms.recalculateRoot()
	return []byte(ms.stateRoot), nil
}

func (ms *MetadataState) recalculateRoot() {
	objIDs := make([]string, 0, len(ms.objects))
	for id := range ms.objects {
		objIDs = append(objIDs, id)
	}
	sort.Strings(objIDs)

	h := sha256.New()
	for _, id := range objIDs {
		h.Write([]byte(id))
		keys := make([]string, 0, len(ms.objects[id]))
		for k := range ms.objects[id] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			h.Write([]byte(k))
			h.Write([]byte(ms.objects[id][k]))
		}
	}
	ms.stateRoot = fmt.Sprintf("%x", h.Sum(nil))
}

func (ms *MetadataState) GetStateRoot() []byte {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return []byte(ms.stateRoot)
}

func (ms *MetadataState) GetMetadata(objectID string) (map[string]string, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	meta, ok := ms.objects[objectID]
	return meta, ok
}

var _ interfaces.StateMachine = (*MetadataState)(nil)
