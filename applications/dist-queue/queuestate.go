package distqueue

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type QueueAction string

const (
	ActionEnqueue QueueAction = "ENQUEUE"
	ActionDequeue QueueAction = "DEQUEUE"
)

type QueuePayload struct {
	Action QueueAction `json:"action"`
	Topic  string      `json:"topic"`
	Data   []byte      `json:"data,omitempty"`
}

type QueueState struct {
	mu        sync.RWMutex
	storage   interfaces.Storage
	queues    map[string][][]byte
	stateRoot string
}

func NewQueueState(store interfaces.Storage) *QueueState {
	return &QueueState{
		storage:   store,
		queues:    make(map[string][][]byte),
		stateRoot: "queue_genesis_root",
	}
}

func (qs *QueueState) Apply(commitLog []byte) ([]byte, error) {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	var payload QueuePayload
	if err := json.Unmarshal(commitLog, &payload); err != nil {
		return nil, fmt.Errorf("dist-queue failed to unmarshal payload: %w", err)
	}

	switch payload.Action {
	case ActionEnqueue:
		qs.queues[payload.Topic] = append(qs.queues[payload.Topic], payload.Data)

	case ActionDequeue:
		q, exists := qs.queues[payload.Topic]
		if !exists || len(q) == 0 {
			return nil, fmt.Errorf("queue for topic %s is empty", payload.Topic)
		}
		qs.queues[payload.Topic] = q[1:]

	default:
		return nil, fmt.Errorf("unsupported queue action: %s", payload.Action)
	}

	qs.recalculateRoot()
	return []byte(qs.stateRoot), nil
}

func (qs *QueueState) recalculateRoot() {
	h := sha256.New()
	for topic, items := range qs.queues {
		h.Write([]byte(topic))
		for _, item := range items {
			h.Write(item)
		}
	}
	qs.stateRoot = fmt.Sprintf("%x", h.Sum(nil))
}

func (qs *QueueState) GetStateRoot() []byte {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	return []byte(qs.stateRoot)
}

func (qs *QueueState) Depth(topic string) int {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	return len(qs.queues[topic])
}

var _ interfaces.StateMachine = (*QueueState)(nil)
