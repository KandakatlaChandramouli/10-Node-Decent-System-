package security

import (
	"sync"
	"time"
)

type Capability string

const (
	CapStateWrite  Capability = "STATE_WRITE"
	CapStateRead   Capability = "STATE_READ"
	CapAdminAccess Capability = "ADMIN_ACCESS"
)

type AuditLogEntry struct {
	Timestamp  time.Time  `json:"timestamp"`
	NodeID     string     `json:"node_id"`
	Capability Capability `json:"capability"`
	Action     string     `json:"action"`
	Allowed    bool       `json:"allowed"`
}

type AuthorizationEngine struct {
	mu           sync.RWMutex
	capabilities map[string]map[Capability]bool
	auditLogs    []AuditLogEntry
}

func NewAuthorizationEngine() *AuthorizationEngine {
	return &AuthorizationEngine{
		capabilities: make(map[string]map[Capability]bool),
		auditLogs:    make([]AuditLogEntry, 0),
	}
}

func (ae *AuthorizationEngine) GrantCapability(nodeID string, cap Capability) {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if _, ok := ae.capabilities[nodeID]; !ok {
		ae.capabilities[nodeID] = make(map[Capability]bool)
	}
	ae.capabilities[nodeID][cap] = true
}

func (ae *AuthorizationEngine) Authorize(nodeID string, cap Capability, action string) bool {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	allowed := false
	if caps, exists := ae.capabilities[nodeID]; exists {
		allowed = caps[cap]
	}

	ae.auditLogs = append(ae.auditLogs, AuditLogEntry{
		Timestamp:  time.Now().UTC(),
		NodeID:     nodeID,
		Capability: cap,
		Action:     action,
		Allowed:    allowed,
	})

	return allowed
}

func (ae *AuthorizationEngine) GetAuditLogs() []AuditLogEntry {
	ae.mu.RLock()
	defer ae.mu.RUnlock()
	return ae.auditLogs
}
