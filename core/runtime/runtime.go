package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sovereignchain "sovereign-chain/applications/sovereign-chain"
	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/types"
	"sovereign-chain/modules/consensus"
	"sovereign-chain/modules/networking"
	"sovereign-chain/modules/storage"
)

type Runtime struct {
	cfg          *types.Config
	storage      interfaces.Storage
	consensus    *consensus.PoWConsensus
	stateMachine *sovereignchain.ChainState
	p2p          *networking.P2PNode
	cancel       context.CancelFunc
}

func New(configPath string) (*Runtime, error) {
	cfg, err := types.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("runtime config error: %w", err)
	}

	var store interfaces.Storage
	switch cfg.Storage.Backend {
	case "pebble":
		pStore, err := storage.NewPebbleStorage(cfg.Storage.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to init pebble storage: %w", err)
		}
		store = pStore
	default:
		store = storage.NewMemoryStorage()
	}

	powEngine := consensus.NewPoWConsensus(cfg.Consensus.Difficulty)
	chainState := sovereignchain.NewChainState(store)
	p2pNode := networking.NewP2PNode(cfg.Node.Port, cfg.P2P.Peers)

	return &Runtime{
		cfg:          cfg,
		storage:      store,
		consensus:    powEngine,
		stateMachine: chainState,
		p2p:          p2pNode,
	}, nil
}

func (r *Runtime) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	if err := r.p2p.Start(ctx); err != nil {
		return fmt.Errorf("failed to start p2p node: %w", err)
	}

	go r.eventLoop(ctx)

	types.Log("INFO", "runtime", fmt.Sprintf("Node online [Port: %d | Storage: %s | Consensus: %s]",
		r.cfg.Node.Port, r.cfg.Storage.Backend, r.cfg.Consensus.Algorithm), r.cfg.Node.ID)
	return nil
}

func (r *Runtime) eventLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var pendingTxs []types.Transaction

	for {
		select {
		case <-ctx.Done():
			return
		case tx := <-r.p2p.GetTxChan():
			pendingTxs = append(pendingTxs, tx)
		case remoteBlock := <-r.p2p.GetBlockChan():
			data, _ := json.Marshal(remoteBlock)
			r.stateMachine.Apply(data)
			types.Log("INFO", "consensus", fmt.Sprintf("Applied remote block #%d: %s", remoteBlock.Index, remoteBlock.Hash), r.cfg.Node.ID)
		case <-ticker.C:
			if len(pendingTxs) == 0 {
				continue
			}
			block := types.Block{
				Index:        1,
				Timestamp:    time.Now(),
				Transactions: pendingTxs,
				PrevHash:     "0000000000000000000000000000000000000000000000000000000000000000",
			}
			if err := r.consensus.MineBlock(ctx, &block); err == nil {
				data, _ := json.Marshal(block)
				r.stateMachine.Apply(data)
				r.p2p.BroadcastBlock(block)
				types.Log("INFO", "consensus", fmt.Sprintf("Mined block #%d: %s", block.Index, block.Hash), r.cfg.Node.ID)
				pendingTxs = nil
			}
		}
	}
}

func (r *Runtime) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}
	return r.storage.Close()
}
