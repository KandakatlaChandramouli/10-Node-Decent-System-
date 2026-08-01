package rwa

import (
	"encoding/json"
	"testing"
)

func TestRWALedger(t *testing.T) {
	ledger := NewLedger()

	validPayload := RWAStateTransition{
		AssetID:      "DUBAI-DIFC-TOWER-1",
		TotalShares:  1000000,
		IssuerPubKey: "0xMinistryOfFinance",
	}
	payloadBytes, _ := json.Marshal(validPayload)

	// TEST 1: Execute Valid RWA Fractionalization
	if err := ledger.Execute(payloadBytes); err != nil {
		t.Fatalf("FAULT: RWA ledger failed to execute valid state transition: %v", err)
	}

	// TEST 2: Prevent Asset Double-Registration
	if err := ledger.Execute(payloadBytes); err == nil {
		t.Fatal("SECURITY FAULT: RWA ledger allowed duplicate asset registration")
	}

	// TEST 3: Verify Internal State
	if ledger.Assets["DUBAI-DIFC-TOWER-1"] != 1000000 {
		t.Fatal("FAULT: Asset shares not correctly recorded in ledger memory")
	}
}
