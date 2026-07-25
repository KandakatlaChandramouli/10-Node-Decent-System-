package interfaces

import "context"

type ExecutionResult struct {
	StateChanges map[string][]byte
	Receipts     [][]byte
	GasUsed      uint64
}

type ExecutionEngine interface {
	Execute(ctx context.Context, payload []byte) (*ExecutionResult, error)
}

type AuthenticatedState interface {
	ApplyChanges(changes map[string][]byte) ([]byte, error)
	GetProof(key []byte) ([][]byte, error)
	VerifyProof(key []byte, proof [][]byte, root []byte) bool
}
