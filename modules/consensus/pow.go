package consensus

import (
	"context"
	"strings"

	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/types"
)

type PoWConsensus struct {
	difficulty int
	commitChan chan []byte
}

func NewPoWConsensus(difficulty int) *PoWConsensus {
	return &PoWConsensus{
		difficulty: difficulty,
		commitChan: make(chan []byte, 100),
	}
}

func (pow *PoWConsensus) MineBlock(ctx context.Context, block *types.Block) error {
	target := strings.Repeat("0", pow.difficulty)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			hash := block.CalculateHash()
			if strings.HasPrefix(hash, target) {
				block.Hash = hash
				return nil
			}
			block.Nonce++
		}
	}
}

func (pow *PoWConsensus) Propose(ctx context.Context, data []byte) error {
	select {
	case pow.commitChan <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pow *PoWConsensus) Commit() <-chan []byte {
	return pow.commitChan
}

var _ interfaces.Consensus = (*PoWConsensus)(nil)
