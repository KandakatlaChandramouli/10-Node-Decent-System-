package execution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type PayloadType string

const (
	CmdTypeRaw PayloadType = "RAW"
)

type Command struct {
	Type    PayloadType `json:"type"`
	Payload []byte      `json:"payload"`
}

type TransactionReceipt struct {
	TxID      string `json:"tx_id"`
	Status    string `json:"status"`
	StateRoot string `json:"state_root"`
	Error     string `json:"error,omitempty"`
}

type CommandHandler func(payload []byte) ([]byte, error)

type ExecutionRouter struct {
	mu          sync.RWMutex
	handlers    map[PayloadType]CommandHandler
	storage     interfaces.Storage
	currentRoot string
	receipts    map[string]TransactionReceipt
}

func NewExecutionRouter(store interfaces.Storage) *ExecutionRouter {
	return &ExecutionRouter{
		handlers:    make(map[PayloadType]CommandHandler),
		storage:     store,
		currentRoot: "exec_router_genesis",
		receipts:    make(map[string]TransactionReceipt),
	}
}

func (er *ExecutionRouter) RegisterHandler(pType PayloadType, handler CommandHandler) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.handlers[pType] = handler
}

func (er *ExecutionRouter) ApplyTransactional(txID string, commitLog []byte) (TransactionReceipt, error) {
	er.mu.Lock()
	defer er.mu.Unlock()

	var cmd Command
	if err := json.Unmarshal(commitLog, &cmd); err != nil {
		receipt := TransactionReceipt{TxID: txID, Status: "FAILED", Error: err.Error()}
		er.receipts[txID] = receipt
		return receipt, fmt.Errorf("unmarshal command error: %w", err)
	}

	handler, exists := er.handlers[cmd.Type]
	if !exists {
		receipt := TransactionReceipt{TxID: txID, Status: "FAILED", Error: "unregistered handler"}
		er.receipts[txID] = receipt
		return receipt, fmt.Errorf("no handler registered for payload type: %s", cmd.Type)
	}

	backupRoot := er.currentRoot
	resultRoot, err := handler(cmd.Payload)
	if err != nil {
		er.currentRoot = backupRoot
		receipt := TransactionReceipt{TxID: txID, Status: "ROLLED_BACK", Error: err.Error()}
		er.receipts[txID] = receipt
		return receipt, fmt.Errorf("handler execution failed, transaction rolled back: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(er.currentRoot))
	h.Write(resultRoot)
	er.currentRoot = fmt.Sprintf("%x", h.Sum(nil))

	receipt := TransactionReceipt{
		TxID:      txID,
		Status:    "COMMITTED",
		StateRoot: er.currentRoot,
	}
	er.receipts[txID] = receipt
	return receipt, nil
}

func (er *ExecutionRouter) Apply(commitLog []byte) ([]byte, error) {
	receipt, err := er.ApplyTransactional("tx_auto_gen", commitLog)
	if err != nil {
		return nil, err
	}
	return []byte(receipt.StateRoot), nil
}

func (er *ExecutionRouter) GetReceipt(txID string) (TransactionReceipt, bool) {
	er.mu.RLock()
	defer er.mu.RUnlock()
	r, ok := er.receipts[txID]
	return r, ok
}

func (er *ExecutionRouter) GetStateRoot() []byte {
	er.mu.RLock()
	defer er.mu.RUnlock()
	return []byte(er.currentRoot)
}

var _ interfaces.StateMachine = (*ExecutionRouter)(nil)
