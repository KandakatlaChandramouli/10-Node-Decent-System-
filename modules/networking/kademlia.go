package networking

import (
	"crypto/sha256"
	"math/big"
	"sync"
	"time"
)

type PeerInfo struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Score     int       `json:"score"`
	LastSeen  time.Time `json:"last_seen"`
	LatencyMs int       `json:"latency_ms"`
}

type KBucket struct {
	mu    sync.RWMutex
	peers []PeerInfo
}

type KademliaDHT struct {
	mu           sync.RWMutex
	localNodeID  string
	buckets      [256]*KBucket
	peerScores   map[string]int
}

func NewKademliaDHT(nodeID string) *KademliaDHT {
	dht := &KademliaDHT{
		localNodeID: nodeID,
		peerScores:  make(map[string]int),
	}
	for i := 0; i < 256; i++ {
		dht.buckets[i] = &KBucket{peers: make([]PeerInfo, 0)}
	}
	return dht
}

func (k *KademliaDHT) CalculateXORDistance(targetID string) *big.Int {
	h1 := sha256.Sum256([]byte(k.localNodeID))
	h2 := sha256.Sum256([]byte(targetID))

	res := make([]byte, 32)
	for i := 0; i < 32; i++ {
		res[i] = h1[i] ^ h2[i]
	}
	return new(big.Int).SetBytes(res)
}

func (k *KademliaDHT) AddPeer(peer PeerInfo) {
	k.mu.Lock()
	defer k.mu.Unlock()

	dist := k.CalculateXORDistance(peer.ID)
	bucketIdx := dist.BitLen() - 1
	if bucketIdx < 0 {
		bucketIdx = 0
	}
	if bucketIdx >= 256 {
		bucketIdx = 255
	}

	bucket := k.buckets[bucketIdx]
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	for i, p := range bucket.peers {
		if p.ID == peer.ID {
			bucket.peers[i] = peer
			return
		}
	}
	if len(bucket.peers) < 20 { // K-bucket capacity limit
		bucket.peers = append(bucket.peers, peer)
	}
	k.peerScores[peer.ID] = peer.Score
}

func (k *KademliaDHT) AdjustPeerScore(peerID string, delta int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.peerScores[peerID] += delta
}

func (k *KademliaDHT) GetPeerScore(peerID string) int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.peerScores[peerID]
}
