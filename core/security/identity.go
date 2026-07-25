package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type NodeIdentity struct {
	NodeID     string
	PublicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func GenerateNodeIdentity() (*NodeIdentity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 keypair: %w", err)
	}

	hash := sha256.Sum256(pub)
	nodeID := "node_" + hex.EncodeToString(hash[:8])

	return &NodeIdentity{
		NodeID:     nodeID,
		PublicKey:  pub,
		privateKey: priv,
	}, nil
}

func (ni *NodeIdentity) SignMessage(msg []byte) []byte {
	return ed25519.Sign(ni.privateKey, msg)
}

func VerifyMessageSignature(pubKey ed25519.PublicKey, msg []byte, sig []byte) bool {
	return ed25519.Verify(pubKey, msg, sig)
}
