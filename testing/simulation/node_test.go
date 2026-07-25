package simulation

import (
	"context"
	"testing"
	"time"

	"sovereign-chain/core/types"
	"sovereign-chain/modules/consensus"
)

func TestProofOfWorkMining(t *testing.T) {
	pow := consensus.NewPoWConsensus(2)
	block := types.Block{
		Index:     1,
		Timestamp: time.Now(),
		PrevHash:  "0000",
		Nonce:     0,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pow.MineBlock(ctx, &block)
	if err != nil {
		t.Fatalf("Mining failed: %v", err)
	}

	if block.Hash == "" {
		t.Fatalf("Expected valid hash, got empty string")
	}

	t.Logf("Mined Hash: %s with Nonce: %d", block.Hash, block.Nonce)
}
