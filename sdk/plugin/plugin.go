package plugin

import (
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type Plugin interface {
	Name() string
	Version() string
	Init(store interfaces.Storage) error
}

type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
	}
}

func (pm *PluginManager) RegisterPlugin(p Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[p.Name()]; exists {
		return fmt.Errorf("plugin %s already registered", p.Name())
	}
	pm.plugins[p.Name()] = p
	return nil
}

func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	p, ok := pm.plugins[name]
	return p, ok
}
