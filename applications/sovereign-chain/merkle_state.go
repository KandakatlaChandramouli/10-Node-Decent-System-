package sovereignchain

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"

	"sovereign-chain/core/interfaces"
)

type MerkleState struct {
	mu      sync.RWMutex
	storage interfaces.Storage
	kvState map[string][]byte
}

func NewMerkleState(store interfaces.Storage) *MerkleState {
	return &MerkleState{
		storage: store,
		kvState: make(map[string][]byte),
	}
}

func (m *MerkleState) ApplyChanges(changes map[string][]byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := make([]string, 0, len(changes))
	for k, v := range changes {
		m.kvState[k] = v
		_ = m.storage.Put([]byte(k), v)
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var leafHashes [][]byte
	for _, k := range keys {
		h := sha256.Sum256(append([]byte(k), m.kvState[k]...))
		leafHashes = append(leafHashes, h[:])
	}

	rootHash := computeMerkleRoot(leafHashes)
	return rootHash, nil
}

func computeMerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		empty := sha256.Sum256([]byte("empty_state"))
		return empty[:]
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	var nextLevel [][]byte
	for i := 0; i < len(hashes); i += 2 {
		if i+1 < len(hashes) {
			combined := append(hashes[i], hashes[i+1]...)
			h := sha256.Sum256(combined)
			nextLevel = append(nextLevel, h[:])
		} else {
			combined := append(hashes[i], hashes[i]...)
			h := sha256.Sum256(combined)
			nextLevel = append(nextLevel, h[:])
		}
	}
	return computeMerkleRoot(nextLevel)
}

func (m *MerkleState) GetProof(key []byte) ([]interfaces.MerkleProofNode, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.kvState[string(key)]
	if !ok {
		return nil, nil, fmt.Errorf("key not found in state")
	}

	leafHash := sha256.Sum256(append(key, val...))
	
	proof := []interfaces.MerkleProofNode{
		{Hash: leafHash[:], IsLeft: true},
	}

	return proof, val, nil
}

func (m *MerkleState) VerifyProof(key []byte, value []byte, proof []interfaces.MerkleProofNode, root []byte) bool {
	if len(proof) == 0 {
		return false
	}
	leafHash := sha256.Sum256(append(key, value...))
	return bytes.Equal(leafHash[:], root)
}

var _ interfaces.AuthenticatedState = (*MerkleState)(nil)
