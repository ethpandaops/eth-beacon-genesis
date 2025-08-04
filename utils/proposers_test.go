package utils

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestGetGenesisProposers(t *testing.T) {
	// Create test config
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":              uint64(32),
		"MAX_EFFECTIVE_BALANCE":        uint64(32_000_000_000),
		"SHUFFLE_ROUND_COUNT":          uint64(90),
		"EPOCHS_PER_HISTORICAL_VECTOR": uint64(65536),
		"MIN_SEED_LOOKAHEAD":           uint64(1),
		"FAR_FUTURE_EPOCH":             uint64(18446744073709551615),
		"DOMAIN_BEACON_PROPOSER":       []byte{0x00, 0x00, 0x00, 0x00},
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	// Create test validators
	validators := make([]*phase0.Validator, 100)
	for i := range validators {
		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey{byte(i)}, // Dummy public key
			ActivationEpoch:  0,                         // Active at genesis
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: phase0.Gwei(32_000_000_000), // 32 ETH
		}
	}

	// Test genesis block hash
	genesisBlockHash := phase0.Hash32{0x01, 0x02, 0x03}

	// Get proposers
	proposers, err := GetGenesisProposers(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("Failed to get genesis proposers: %v", err)
	}

	// Verify we got proposers for 2 epochs (64 slots with 32 slots per epoch)
	expectedSlots := 64
	if len(proposers) != expectedSlots {
		t.Errorf("Expected %d proposers, got %d", expectedSlots, len(proposers))
	}

	// Verify all proposers are valid indices
	for i, proposer := range proposers {
		if int(proposer) >= len(validators) {
			t.Errorf("Invalid proposer index %d at slot %d", proposer, i)
		}
	}

	// Verify proposers are deterministic
	proposers2, err := GetGenesisProposers(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("Failed to get genesis proposers second time: %v", err)
	}

	for i := range proposers {
		if proposers[i] != proposers2[i] {
			t.Errorf("Proposer mismatch at slot %d: %d vs %d", i, proposers[i], proposers2[i])
		}
	}
}

func TestGetGenesisProposersNoActiveValidators(t *testing.T) {
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH": uint64(32),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	// Create validators that are not active at genesis
	validators := make([]*phase0.Validator, 10)
	for i := range validators {
		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey{byte(i)},
			ActivationEpoch:  1, // Not active at genesis
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: phase0.Gwei(32_000_000_000),
		}
	}

	genesisBlockHash := phase0.Hash32{0x01, 0x02, 0x03}

	_, err := GetGenesisProposers(clConfig, validators, genesisBlockHash)
	if err == nil {
		t.Error("Expected error for no active validators, got nil")
	}
}

func TestGetGenesisProposersMainnetExample(t *testing.T) {
	// Create mainnet config
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":               uint64(32),
		"MAX_EFFECTIVE_BALANCE":         uint64(32_000_000_000),
		"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
		"SHUFFLE_ROUND_COUNT":           uint64(90),
		"EPOCHS_PER_HISTORICAL_VECTOR":  uint64(65536),
		"MIN_SEED_LOOKAHEAD":            uint64(1),
		"FAR_FUTURE_EPOCH":              uint64(18446744073709551615),
		"DOMAIN_BEACON_PROPOSER":        []byte{0x00, 0x00, 0x00, 0x00},
		"ELECTRA_FORK_EPOCH":            uint64(0), // Electra active at genesis for Fulu
	}
	clConfig := createTestConfig(t, "mainnet", configValues)

	// Create 100 validators with 32 ETH each
	validators := make([]*phase0.Validator, 100)
	for i := range validators {
		pubKey := make([]byte, 48)
		// Create a unique public key for each validator
		pubKey[0] = byte(i)
		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey(pubKey),
			ActivationEpoch:  0, // Active at genesis
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: phase0.Gwei(32_000_000_000), // 32 ETH
		}
	}

	// Use the provided genesis block hash (0x60c93f6d06bb60444099b54175f11a1d8b9d91b1280fdb7e64335602a968fabe)
	genesisBlockHash := phase0.Hash32{
		0x60, 0xc9, 0x3f, 0x6d, 0x06, 0xbb, 0x60, 0x44, 0x40, 0x99, 0xb5, 0x41, 0x75, 0xf1, 0x1a, 0x1d,
		0x8b, 0x9d, 0x91, 0xb1, 0x28, 0x0f, 0xdb, 0x7e, 0x64, 0x33, 0x56, 0x02, 0xa9, 0x68, 0xfa, 0xbe,
	}

	// Get proposers
	proposers, err := GetGenesisProposers(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("Failed to get genesis proposers: %v", err)
	}

	// Expected proposers for epoch 0 (slots 0-31)
	expectedProposers := map[uint64]phase0.ValidatorIndex{
		0:  72,
		1:  85,
		2:  26,
		3:  2,
		4:  42,
		5:  84,
		6:  19,
		7:  25,
		8:  65,
		9:  85,
		10: 29,
		11: 96,
		12: 36,
		13: 33,
		14: 53,
		15: 82,
		16: 94,
		17: 31,
		18: 47,
		19: 24,
		20: 72,
		21: 90,
		22: 79,
		23: 75,
		24: 15,
		25: 72,
		26: 51,
		27: 46,
		28: 7,
		29: 52,
		30: 67,
		31: 3,
		32: 71,
		63: 30,
	}

	// Verify the expected proposers
	for slot, expectedProposer := range expectedProposers {
		if slot >= uint64(len(proposers)) {
			t.Errorf("Slot %d is out of range", slot)
			continue
		}

		actualProposer := proposers[slot]
		if actualProposer != expectedProposer {
			t.Errorf("Slot %d: expected proposer %d, got %d", slot, expectedProposer, actualProposer)
		}
	}

	// Also verify we have proposers for epoch 1 (slots 32-63)
	if len(proposers) < 64 {
		t.Errorf("Expected proposers for 2 epochs (64 slots), got %d", len(proposers))
	}
}
