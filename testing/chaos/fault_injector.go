package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type FaultType int

const (
	FaultNetworkPartition FaultType = iota
	FaultMessageDelay
	FaultNodeCrash
	FaultStorageLatency
)

type FaultConfig struct {
	PartitionRatio   float64
	MaxDelayMs       int
	CrashProbability float64
}

type FaultInjector struct {
	mu            sync.RWMutex
	cfg           FaultConfig
	partitioned   map[string]bool
	crashedNodes  map[string]bool
	rnd           *rand.Rand
}

func NewFaultInjector(cfg FaultConfig) *FaultInjector {
	return &FaultInjector{
		cfg:          cfg,
		partitioned:  make(map[string]bool),
		crashedNodes: make(map[string]bool),
		rnd:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (f *FaultInjector) InjectPartition(nodeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.partitioned[nodeID] = true
}

func (f *FaultInjector) HealPartition(nodeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.partitioned, nodeID)
}

func (f *FaultInjector) IsPartitioned(nodeID string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.partitioned[nodeID]
}

func (f *FaultInjector) SimulateMessageLatency(ctx context.Context) error {
	if f.cfg.MaxDelayMs <= 0 {
		return nil
	}
	f.mu.RLock()
	delay := f.rnd.Intn(f.cfg.MaxDelayMs)
	f.mu.RUnlock()

	select {
	case <-time.After(time.Duration(delay) * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *FaultInjector) ShouldDropMessage(nodeID string) bool {
	if f.IsPartitioned(nodeID) {
		return true
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.rnd.Float64() < f.cfg.PartitionRatio
}

func (f *FaultInjector) ExecuteNodeCrash(nodeID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.crashedNodes[nodeID] = true
	return fmt.Errorf("simulated crash executed for node: %s", nodeID)
}

func (f *FaultInjector) IsNodeCrashed(nodeID string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.crashedNodes[nodeID]
}
