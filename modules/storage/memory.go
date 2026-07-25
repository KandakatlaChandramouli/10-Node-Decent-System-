package storage

import (
	"fmt"
	"sync"
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

func (m *MemoryStorage) Put(key, value []byte) error {
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
		return nil, fmt.Errorf("key not found")
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
	return nil
}
