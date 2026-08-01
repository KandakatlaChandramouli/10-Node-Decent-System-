package privacy

import (
	"context"
	"fmt"
	"time"
)

// PeerNetwork defines the interface to the underlying LibP2P routing layer
type PeerNetwork interface {
	RequestPayload(ctx context.Context, txID string, jurisdiction string) ([]byte, error)
}

type SideChannelFetcher struct {
	network PeerNetwork
	vault   *JurisdictionVault
	timeout time.Duration
}

func NewSideChannelFetcher(net PeerNetwork, vault *JurisdictionVault) *SideChannelFetcher {
	return &SideChannelFetcher{
		network: net,
		vault:   vault,
		timeout: 5 * time.Second, // Max wait before failing the state machine apply
	}
}

// FetchAndStore blocks until the payload is retrieved from the P2P mesh and verified, or times out.
func (f *SideChannelFetcher) FetchAndStore(txID string, expectedHash [32]byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	fmt.Printf("[PDC Network] Missing payload %s. Fetching from authorized peers in %s...\n", txID, f.vault.NodeJurisdiction)

	encryptedData, err := f.network.RequestPayload(ctx, txID, f.vault.NodeJurisdiction)
	if err != nil {
		return fmt.Errorf("failed to retrieve side-channel payload from peers: %w", err)
	}

	payload := PrivatePayload{
		TxID:             txID,
		JurisdictionCode: f.vault.NodeJurisdiction,
		EncryptedData:    encryptedData,
	}

	// Route through the vault to ensure strict jurisdiction enforcement
	if err := f.vault.StoreIfAuthorized(payload); err != nil {
		return err
	}

	// Cryptographically verify the fetched bytes against the Raft anchor
	_, err = f.vault.RetrieveAndVerify(txID, expectedHash)
	return err
}
