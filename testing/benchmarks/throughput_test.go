package benchmarks

import (
	"context"
	"testing"
	"time"

	"sovereign-chain/core/types"
	"sovereign-chain/modules/consensus"
)

func BenchmarkPoWMiningDifficulty2(b *testing.B) {
	pow := consensus.NewPoWConsensus(2)
	block := types.Block{
		Index:     1,
		Timestamp: time.Now(),
		PrevHash:  "00000000",
		Nonce:     0,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.Nonce = i * 1000
		_ = pow.MineBlock(ctx, &block)
	}
}

func BenchmarkPoWMiningDifficulty4(b *testing.B) {
	pow := consensus.NewPoWConsensus(4)
	block := types.Block{
		Index:     1,
		Timestamp: time.Now(),
		PrevHash:  "00000000",
		Nonce:     0,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		block.Nonce = i * 1000
		_ = pow.MineBlock(ctx, &block)
	}
}
