package execution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type CommandType string

const (
	CmdTypeRaw PayloadType = "RAW"
)

type PayloadType string

type Command struct {
	Type    PayloadType `json:"type"`
	Payload []byte      `json:"payload"`
}

type CommandHandler func(payload []byte) ([]byte, error)

type ExecutionRouter struct {
	mu          sync.RWMutex
	handlers    map[PayloadType]CommandHandler
	storage     interfaces.Storage
	currentRoot string
}

func NewExecutionRouter(store interfaces.Storage) *ExecutionRouter {
	return &ExecutionRouter{
		handlers:    make(map[PayloadType]CommandHandler),
		storage:     store,
		currentRoot: "exec_router_genesis",
	}
}

func (er *ExecutionRouter) RegisterHandler(pType PayloadType, handler CommandHandler) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.handlers[pType] = handler
}

func (er *ExecutionRouter) Apply(commitLog []byte) ([]byte, error) {
	er.mu.Lock()
	defer er.mu.Unlock()

	var cmd Command
	if err := json.Unmarshal(commitLog, &cmd); err != nil {
		return nil, fmt.Errorf("execution router failed to unmarshal command: %w", err)
	}

	handler, exists := er.handlers[cmd.Type]
	if !exists {
		return nil, fmt.Errorf("no handler registered for payload type: %s", cmd.Type)
	}

	resultRoot, err := handler(cmd.Payload)
	if err != nil {
		return nil, fmt.Errorf("handler execution failed for type %s: %w", cmd.Type, err)
	}

	h := sha256.New()
	h.Write([]byte(er.currentRoot))
	h.Write(resultRoot)
	er.currentRoot = fmt.Sprintf("%x", h.Sum(nil))

	return []byte(er.currentRoot), nil
}

func (er *ExecutionRouter) GetStateRoot() []byte {
	er.mu.RLock()
	defer er.mu.RUnlock()
	return []byte(er.currentRoot)
}

var _ interfaces.StateMachine = (*ExecutionRouter)(nil)
