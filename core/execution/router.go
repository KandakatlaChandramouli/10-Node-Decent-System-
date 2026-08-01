package execution

import (
	"crypto/sha256"
	"fmt"
	"sovereign-chain/modules/privacy"
	"strings"
)

type SovereignExecutionRouter struct {
	Vault        *privacy.JurisdictionVault
	Fetcher      *privacy.SideChannelFetcher
	Telemetry    *privacy.TelemetryPipeline
	GlobalState  map[string][32]byte
	PrivateState map[string][]byte
}

func NewSovereignExecutionRouter(vault *privacy.JurisdictionVault, fetcher *privacy.SideChannelFetcher, telemetry *privacy.TelemetryPipeline) *SovereignExecutionRouter {
	return &SovereignExecutionRouter{
		Vault:        vault,
		Fetcher:      fetcher,
		Telemetry:    telemetry,
		GlobalState:  make(map[string][32]byte),
		PrivateState: make(map[string][]byte),
	}
}

// Apply is called sequentially by the Raft consensus engine
func (r *SovereignExecutionRouter) Apply(anchor privacy.AnchoredTransaction) error {
	r.GlobalState[anchor.TxID] = anchor.PayloadHash

	if anchor.JurisdictionCode != r.Vault.NodeJurisdiction {
		return nil
	}

	payloadData, err := r.Vault.RetrieveAndVerify(anchor.TxID, anchor.PayloadHash)

	if err != nil {
		if strings.Contains(err.Error(), "PAYLOAD_MISSING") && r.Fetcher != nil {
			fetchErr := r.Fetcher.FetchAndStore(anchor.TxID, anchor.PayloadHash)
			if fetchErr != nil {
				return fmt.Errorf("CRITICAL FAULT: execution stalled, payload unrecoverable: %v", fetchErr)
			}
			payloadData, _ = r.Vault.RetrieveAndVerify(anchor.TxID, anchor.PayloadHash)
		} else {
			return fmt.Errorf("execution stalled: %v", err)
		}
	}

	// Capture pre-state for ZK trace
	preStateHash := sha256.Sum256(r.PrivateState[anchor.TxID])

	// 5. Apply the private data to the local isolated state
	r.PrivateState[anchor.TxID] = payloadData

	// Capture post-state for ZK trace
	postStateHash := sha256.Sum256(r.PrivateState[anchor.TxID])

	// 6. Emit trace to the async ZK Telemetry Pipeline (Non-blocking)
	if r.Telemetry != nil {
		trace := privacy.ExecutionTrace{
			TxID:             anchor.TxID,
			JurisdictionCode: anchor.JurisdictionCode,
			PreStateRoot:     preStateHash,
			PostStateRoot:    postStateHash,
			PayloadHash:      anchor.PayloadHash,
		}
		select {
		case r.Telemetry.TraceQueue <- trace:
			// Trace successfully buffered
		default:
			fmt.Printf("WARNING: ZK Telemetry queue saturated, dropping trace for %s\n", anchor.TxID)
		}
	}

	fmt.Printf("[Success] Executed isolated state transition for Tx: %s\n", anchor.TxID)
	return nil
}
