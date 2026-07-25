package runtime

import (
	"fmt"
	"sync"

	"sovereign-chain/core/interfaces"
)

type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]interfaces.Service
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]interfaces.Service),
	}
}

func (r *ServiceRegistry) Register(svc interfaces.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[svc.Name()]; exists {
		return fmt.Errorf("service already registered: %s", svc.Name())
	}
	r.services[svc.Name()] = svc
	return nil
}

func (r *ServiceRegistry) Get(name string) (interfaces.Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[name]
	return svc, ok
}

func (r *ServiceRegistry) HealthCheckAll() map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	healthMap := make(map[string]error)
	for name, svc := range r.services {
		healthMap[name] = svc.Health()
	}
	return healthMap
}
