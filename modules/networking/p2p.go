package networking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"sovereign-chain/core/interfaces"
	"sovereign-chain/core/types"
)

type P2PNode struct {
	mu        sync.RWMutex
	port      int
	peers     []string
	host      host.Host
	pubsub    *pubsub.PubSub
	topic     *pubsub.Topic
	sub       *pubsub.Subscription
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
	return "Networking-LibP2P"
}

func (p *P2PNode) Dependencies() []string {
	return []string{"Storage-Backend"}
}

func (p *P2PNode) Init(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.host != nil {
		return nil
	}

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p.port+1000)),
	)
	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %w", err)
	}
	p.host = h

	ps, err := pubsub.NewGossipSub(ctx, p.host)
	if err != nil {
		return fmt.Errorf("failed to create GossipSub: %w", err)
	}
	p.pubsub = ps

	topic, err := ps.Join("sovereign-blocks")
	if err != nil {
		return fmt.Errorf("failed to join gossip topic: %w", err)
	}
	p.topic = topic

	sub, err := topic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}
	p.sub = sub

	go p.listenGossip(ctx)

	return nil
}

func (p *P2PNode) listenGossip(ctx context.Context) {
	for {
		msg, err := p.sub.Next(ctx)
		if err != nil {
			return
		}
		if msg.ReceivedFrom == p.host.ID() {
			continue
		}
		var block types.Block
		if err := json.Unmarshal(msg.Data, &block); err == nil {
			p.blockChan <- block
		}
	}
}

func (p *P2PNode) Start(ctx context.Context) error {
	if err := p.Init(ctx); err != nil {
		return fmt.Errorf("auto-init failed in Start: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/tx", p.handleTx)
	mux.HandleFunc("/block", p.handleBlock)
	mux.Handle("/metrics", types.PrometheusHandler())
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		peerID := "unknown"
		if p.host != nil {
			peerID = p.host.ID().String()
		}
		fmt.Fprintf(w, "pong [peer_id: %s]", peerID)
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
			types.Log("ERROR", "networking", fmt.Sprintf("HTTP server error: %v", err), "")
		}
	}()

	peerID := "unknown"
	if p.host != nil {
		peerID = p.host.ID().String()
	}

	types.Log("INFO", "networking", fmt.Sprintf("LibP2P host online [ID: %s | Port: %d]", peerID, p.port), "")
	return nil
}

func (p *P2PNode) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.sub != nil {
		p.sub.Cancel()
	}
	if p.host != nil {
		_ = p.host.Close()
	}
	if p.server != nil {
		return p.server.Shutdown(ctx)
	}
	return nil
}

func (p *P2PNode) Health() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.host == nil || len(p.host.Addrs()) == 0 {
		return fmt.Errorf("libp2p host unhealthy")
	}
	return nil
}

func (p *P2PNode) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var tx types.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
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
	var block types.Block
	if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
		http.Error(w, "Invalid block payload", http.StatusBadRequest)
		return
	}
	p.blockChan <- block
	w.WriteHeader(http.StatusAccepted)
}

func (p *P2PNode) BroadcastBlock(block types.Block) {
	data, err := json.Marshal(block)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if p.topic != nil {
		_ = p.topic.Publish(ctx, data)
	}
}

func (p *P2PNode) GetTxChan() <-chan types.Transaction {
	return p.txChan
}

func (p *P2PNode) GetBlockChan() <-chan types.Block {
	return p.blockChan
}

var _ interfaces.DependentService = (*P2PNode)(nil)
