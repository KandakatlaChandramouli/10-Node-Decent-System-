package execution

import (
	"fmt"
	"sovereign-chain/modules/privacy"
)

type SovereignExecutionRouter struct {
	Vault        *privacy.JurisdictionVault
	GlobalState  map[string][32]byte // Merkle-authenticated public state
	PrivateState map[string][]byte  // Jurisdiction-isolated local state
}

func NewSovereignExecutionRouter(vault *privacy.JurisdictionVault) *SovereignExecutionRouter {
	return &SovereignExecutionRouter{
		Vault:        vault,
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
		// Log continuous, but state execution is skipped. We are done.
		fmt.Printf("Node bypassed execution for foreign jurisdiction: %s\n", anchor.JurisdictionCode)
		return nil
	}

	// 3. We are authorized. Fetch the payload from the side-channel vault.
	payloadData, err := r.Vault.RetrieveAndVerify(anchor.TxID, anchor.PayloadHash)
	if err != nil {
		// In a production system, this would trigger a blocking P2P request 
		// to fetch the missing payload from authorized peers before continuing.
		return fmt.Errorf("execution stalled: %v", err)
	}

	// 4. Apply the private data to the local isolated state
	r.PrivateState[anchor.TxID] = payloadData
	fmt.Printf("Successfully executed local state transition for Tx: %s\n", anchor.TxID)
	
	return nil
}
