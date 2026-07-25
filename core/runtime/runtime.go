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
	stateMachine interfaces.StateMachine
	p2p          *networking.P2PNode
	services     []interfaces.Service
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

	rt := &Runtime{
		cfg:          cfg,
		storage:      store,
		consensus:    powEngine,
		stateMachine: chainState,
		p2p:          p2pNode,
		services:     make([]interfaces.Service, 0),
	}

	rt.RegisterService(p2pNode)

	return rt, nil
}

func (r *Runtime) RegisterService(svc interfaces.Service) {
	r.services = append(r.services, svc)
}

func (r *Runtime) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	for _, svc := range r.services {
		if depSvc, ok := svc.(interfaces.DependentService); ok {
			for _, dep := range depSvc.Dependencies() {
				types.Log("INFO", "runtime", fmt.Sprintf("Service [%s] resolved dependency: %s", svc.Name(), dep), r.cfg.Node.ID)
			}
		}

		if err := svc.Init(ctx); err != nil {
			return fmt.Errorf("service %s init failed: %w", svc.Name(), err)
		}
		if err := svc.Start(ctx); err != nil {
			return fmt.Errorf("service %s start failed: %w", svc.Name(), err)
		}
		types.Log("INFO", "runtime", fmt.Sprintf("Started service: %s", svc.Name()), r.cfg.Node.ID)
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
			root, _ := r.stateMachine.Apply(data)
			types.Log("INFO", "consensus", fmt.Sprintf("Applied remote block #%d. State Root: %s", remoteBlock.Index, string(root)), r.cfg.Node.ID)
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
				root, _ := r.stateMachine.Apply(data)
				r.p2p.BroadcastBlock(block)
				types.BlocksMinedTotal.Inc()
				types.Log("INFO", "consensus", fmt.Sprintf("Mined block #%d: %s. New State Root: %s", block.Index, block.Hash, string(root)), r.cfg.Node.ID)
				pendingTxs = nil
			} else {
				types.BlocksFailedTotal.Inc()
			}
		}
	}
}

func (r *Runtime) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if r.cancel != nil {
		r.cancel()
	}

	for i := len(r.services) - 1; i >= 0; i-- {
		svc := r.services[i]
		if err := svc.Stop(ctx); err != nil {
			types.Log("ERROR", "runtime", fmt.Sprintf("Failed stopping service %s: %v", svc.Name(), err), r.cfg.Node.ID)
		}
	}

	return r.storage.Close()
}
