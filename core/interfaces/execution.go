package interfaces

import "context"

type ExecutionResult struct {
	ReadSet       map[string][]byte
	WriteSet      map[string][]byte
	Receipts      [][]byte
	GasUsed       uint64
	ExecutionTime int64
}

type ExecutionEngine interface {
	Execute(ctx context.Context, payload []byte) (*ExecutionResult, error)
}

type MerkleProofNode struct {
	Hash   []byte
	IsLeft bool
}

type AuthenticatedState interface {
	ApplyChanges(changes map[string][]byte) ([]byte, error)
	GetProof(key []byte) ([]MerkleProofNode, []byte, error)
	VerifyProof(key []byte, value []byte, proof []MerkleProofNode, root []byte) bool
}
