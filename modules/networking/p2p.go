package networking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/types"
)

type P2PNode struct {
	mu        sync.RWMutex
	port      int
	peers     []string
	server    *http.Server
	blockChan chan types.Block
	txChan    chan types.Transaction
}

func NewP2PNode(port int, peers []string) *P2PNode {
	return &P2PNode{
		port:      port,
		peers:     peers,
		blockChan: make(chan types.Block, 100),
		txChan:    make(chan types.Transaction, 1000),
	}
}

func (p *P2PNode) Name() string {
	return "Networking-P2P"
}

func (p *P2PNode) Dependencies() []string {
	return []string{"Storage-Backend"}
}

func (p *P2PNode) Init(ctx context.Context) error {
	return nil
}

func (p *P2PNode) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/tx", p.handleTx)
	mux.HandleFunc("/block", p.handleBlock)
	mux.Handle("/metrics", types.PrometheusHandler())
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		p.server.Shutdown(context.Background())
	}()

	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			types.Log("ERROR", "networking", fmt.Sprintf("P2P server error: %v", err), "")
		}
	}()

	types.Log("INFO", "networking", fmt.Sprintf("P2P listening on port %d with %d seed peers", p.port, len(p.peers)), "")
	return nil
}

func (p *P2PNode) Stop(ctx context.Context) error {
	if p.server != nil {
		return p.server.Shutdown(ctx)
	}
	return nil
}

func (p *P2PNode) Health() error {
	return nil
}

func (p *P2PNode) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var tx types.Transaction
	if err := json.Unmarshal(body, &tx); err != nil {
		http.Error(w, "Invalid transaction payload", http.StatusBadRequest)
		return
	}
	types.TxProcessedTotal.Inc()
	p.txChan <- tx
	w.WriteHeader(http.StatusAccepted)
}

func (p *P2PNode) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var block types.Block
	if err := json.Unmarshal(body, &block); err != nil {
		http.Error(w, "Invalid block payload", http.StatusBadRequest)
		return
	}
	p.blockChan <- block
	w.WriteHeader(http.StatusAccepted)
}

func (p *P2PNode) BroadcastBlock(block types.Block) {
	data, _ := json.Marshal(block)
	p.mu.RLock()
	defer p.mu.RUnlock()

	client := http.Client{Timeout: 2 * time.Second}
	for _, peer := range p.peers {
		go func(peer string) {
			resp, err := client.Post(fmt.Sprintf("http://%s/block", peer), "application/json", bytes.NewBuffer(data))
			if err == nil {
				resp.Body.Close()
			}
		}(peer)
	}
}

func (p *P2PNode) GetTxChan() <-chan types.Transaction {
	return p.txChan
}

func (p *P2PNode) GetBlockChan() <-chan types.Block {
	return p.blockChan
}

var _ interfaces.DependentService = (*P2PNode)(nil)
