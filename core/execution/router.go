package execution

import (
	"fmt"
	"strings"
	"sovereign-chain/modules/privacy"
)

type SovereignExecutionRouter struct {
	Vault        *privacy.JurisdictionVault
	Fetcher      *privacy.SideChannelFetcher
	GlobalState  map[string][32]byte // Merkle-authenticated public state
	PrivateState map[string][]byte  // Jurisdiction-isolated local state
}

func NewSovereignExecutionRouter(vault *privacy.JurisdictionVault, fetcher *privacy.SideChannelFetcher) *SovereignExecutionRouter {
	return &SovereignExecutionRouter{
		Vault:        vault,
		Fetcher:      fetcher,
		GlobalState:  make(map[string][32]byte),
		PrivateState: make(map[string][]byte),
	}
}

// Apply is called sequentially by the Raft consensus engine
func (r *SovereignExecutionRouter) Apply(anchor privacy.AnchoredTransaction) error {
	// 1. ALL nodes globally record the proof-of-existence to maintain Raft continuity
	r.GlobalState[anchor.TxID] = anchor.PayloadHash

	// 2. Branch execution: Are we in the authorized jurisdiction?
	if anchor.JurisdictionCode != r.Vault.NodeJurisdiction {
		fmt.Printf("[Bypass] Execution skipped for foreign jurisdiction: %s\n", anchor.JurisdictionCode)
		return nil
	}

	// 3. We are authorized. Fetch the payload from the side-channel vault.
	payloadData, err := r.Vault.RetrieveAndVerify(anchor.TxID, anchor.PayloadHash)
	
	if err != nil {
		// 4. Handle asynchronous race condition: Payload hasn't arrived via P2P yet
		if strings.Contains(err.Error(), "PAYLOAD_MISSING") && r.Fetcher != nil {
			fetchErr := r.Fetcher.FetchAndStore(anchor.TxID, anchor.PayloadHash)
			if fetchErr != nil {
				return fmt.Errorf("CRITICAL FAULT: execution stalled, payload unrecoverable: %v", fetchErr)
			}
			// Retry local read after successful network fetch
			payloadData, _ = r.Vault.RetrieveAndVerify(anchor.TxID, anchor.PayloadHash)
		} else {
			return fmt.Errorf("execution stalled: %v", err)
		}
	}

	// 5. Apply the private data to the local isolated state
	r.PrivateState[anchor.TxID] = payloadData
	fmt.Printf("[Success] Executed isolated state transition for Tx: %s\n", anchor.TxID)
	
	return nil
}
