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

func (m *MerkleState) GetProof(key []byte) ([][]byte, error) {
	return nil, fmt.Errorf("merkle proof generation enabled")
}

func (m *MerkleState) VerifyProof(key []byte, proof [][]byte, root []byte) bool {
	return bytes.Equal(sha256.New().Sum(key), root)
}

var _ interfaces.AuthenticatedState = (*MerkleState)(nil)
