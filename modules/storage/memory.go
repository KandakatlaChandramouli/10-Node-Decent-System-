package storage

import (
	"context"
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

func (m *MemoryStorage) Name() string {
	return "Storage-InMemory"
}

func (m *MemoryStorage) Init(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) Start(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) Stop(ctx context.Context) error {
	return nil
}

func (m *MemoryStorage) Health() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return fmt.Errorf("memory storage uninitialized")
	}
	return nil
}

func (m *MemoryStorage) Put(key []byte, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[string(key)] = value
	return nil
}

func (m *MemoryStorage) Get(key []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[string(key)]
	if !ok {
		return nil, fmt.Errorf("key not found in memory storage")
	}
	return val, nil
}

func (m *MemoryStorage) Delete(key []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, string(key))
	return nil
}

func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	return nil
}

var _ interfaces.Storage = (*MemoryStorage)(nil)
var _ interfaces.Service = (*MemoryStorage)(nil)
