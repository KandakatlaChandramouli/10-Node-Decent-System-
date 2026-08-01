package privacy

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"sync"
)

// AnchoredTransaction is the ONLY thing that goes into the Raft Log
type AnchoredTransaction struct {
	TxID             string
	JurisdictionCode string
	PayloadHash      [32]byte
	Timestamp        int64
}

// PrivatePayload is transmitted via P2P side-channels (NOT Raft)
type PrivatePayload struct {
	TxID             string
	JurisdictionCode string
	EncryptedData    []byte
}

type JurisdictionVault struct {
	mu               sync.RWMutex
	NodeJurisdiction string
	sideDB           map[string]PrivatePayload // Transient memory or encrypted disk
}

func NewJurisdictionVault(jurisdiction string) *JurisdictionVault {
	return &JurisdictionVault{
		NodeJurisdiction: jurisdiction,
		sideDB:           make(map[string]PrivatePayload),
	}
}

// StoreIfAuthorized prevents foreign data from ever being written to disk
func (v *JurisdictionVault) StoreIfAuthorized(payload PrivatePayload) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if payload.JurisdictionCode != v.NodeJurisdiction {
		return errors.New("JURISDICTION_VIOLATION: Node is legally barred from holding this data")
	}

	v.sideDB[payload.TxID] = payload
	return nil
}

// RetrieveAndVerify fetches the payload and ensures it matches the global Raft consensus hash
func (v *JurisdictionVault) RetrieveAndVerify(txID string, expectedHash [32]byte) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	payload, exists := v.sideDB[txID]
	if !exists {
		return nil, errors.New("PAYLOAD_MISSING: Authorized node lacks side-channel data")
	}

	actualHash := sha256.Sum256(payload.EncryptedData)
	if !bytes.Equal(actualHash[:], expectedHash[:]) {
		return nil, errors.New("CRYPTOGRAPHIC_FAULT: Side-channel payload does not match Raft anchor")
	}

	return payload.EncryptedData, nil
}
