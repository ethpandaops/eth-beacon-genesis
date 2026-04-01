package beaconutils

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestGetGenesisPTCWindow(t *testing.T) {
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":               uint64(8),
		"MIN_SEED_LOOKAHEAD":            uint64(1),
		"PTC_SIZE":                      uint64(4),
		"SHUFFLE_ROUND_COUNT":           uint64(10),
		"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
		"TARGET_COMMITTEE_SIZE":         uint64(4),
		"MAX_COMMITTEES_PER_SLOT":       uint64(4),
		"DOMAIN_PTC_ATTESTER":           []byte{0x0c, 0x00, 0x00, 0x00},
		"DOMAIN_BEACON_ATTESTER":        []byte{0x01, 0x00, 0x00, 0x00},
		"ELECTRA_FORK_EPOCH":            uint64(0),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	validators := make([]*phase0.Validator, 64)
	for i := range validators {
		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey{byte(i)},
			ActivationEpoch:  0,
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: phase0.Gwei(32_000_000_000),
		}
	}

	genesisBlockHash := phase0.Hash32{0x01, 0x02, 0x03}

	ptcWindow, err := GetGenesisPTCWindow(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// (2 + MIN_SEED_LOOKAHEAD) * SLOTS_PER_EPOCH = 3 * 8 = 24
	expectedSlots := uint64(24)
	if uint64(len(ptcWindow)) != expectedSlots {
		t.Fatalf("wrong PTC window length: got %d, want %d", len(ptcWindow), expectedSlots)
	}

	ptcSize := uint64(4)

	// Check all entries have correct PTC_SIZE
	for i, entry := range ptcWindow {
		if uint64(len(entry)) != ptcSize {
			t.Errorf("slot %d: wrong PTC size: got %d, want %d", i, len(entry), ptcSize)
		}
	}

	// First SLOTS_PER_EPOCH entries must be all zeros (empty previous epoch)
	for i := 0; i < 8; i++ {
		for j, idx := range ptcWindow[i] {
			if idx != 0 {
				t.Errorf("empty epoch slot %d pos %d: expected 0, got %d", i, j, idx)
			}
		}
	}

	// Remaining entries should contain valid validator indices
	for i := 8; i < int(expectedSlots); i++ {
		for j, idx := range ptcWindow[i] {
			if uint64(idx) >= uint64(len(validators)) {
				t.Errorf("slot %d pos %d: invalid validator index %d", i, j, idx)
			}
		}
	}
}

func TestGetGenesisPTCWindowDeterminism(t *testing.T) {
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":               uint64(8),
		"MIN_SEED_LOOKAHEAD":            uint64(1),
		"PTC_SIZE":                      uint64(4),
		"SHUFFLE_ROUND_COUNT":           uint64(10),
		"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
		"TARGET_COMMITTEE_SIZE":         uint64(4),
		"MAX_COMMITTEES_PER_SLOT":       uint64(4),
		"DOMAIN_PTC_ATTESTER":           []byte{0x0c, 0x00, 0x00, 0x00},
		"DOMAIN_BEACON_ATTESTER":        []byte{0x01, 0x00, 0x00, 0x00},
		"ELECTRA_FORK_EPOCH":            uint64(0),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	validators := make([]*phase0.Validator, 64)
	for i := range validators {
		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey{byte(i)},
			ActivationEpoch:  0,
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: phase0.Gwei(32_000_000_000),
		}
	}

	genesisBlockHash := phase0.Hash32{0x01, 0x02, 0x03}

	ptcWindow1, err := GetGenesisPTCWindow(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	ptcWindow2, err := GetGenesisPTCWindow(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	for i := range ptcWindow1 {
		for j := range ptcWindow1[i] {
			if ptcWindow1[i][j] != ptcWindow2[i][j] {
				t.Errorf("non-deterministic at slot %d pos %d: %d vs %d",
					i, j, ptcWindow1[i][j], ptcWindow2[i][j])
			}
		}
	}
}

func TestGetGenesisPTCWindowNoActiveValidators(t *testing.T) {
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":               uint64(8),
		"MIN_SEED_LOOKAHEAD":            uint64(1),
		"PTC_SIZE":                      uint64(4),
		"SHUFFLE_ROUND_COUNT":           uint64(10),
		"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
		"TARGET_COMMITTEE_SIZE":         uint64(4),
		"MAX_COMMITTEES_PER_SLOT":       uint64(4),
		"DOMAIN_PTC_ATTESTER":           []byte{0x0c, 0x00, 0x00, 0x00},
		"DOMAIN_BEACON_ATTESTER":        []byte{0x01, 0x00, 0x00, 0x00},
		"ELECTRA_FORK_EPOCH":            uint64(0),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

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

	_, err := GetGenesisPTCWindow(clConfig, validators, genesisBlockHash)
	if err == nil {
		t.Error("expected error for no active validators, got nil")
	}
}

func TestGetGenesisPTCWindowBalanceWeighting(t *testing.T) {
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":               uint64(8),
		"MIN_SEED_LOOKAHEAD":            uint64(1),
		"PTC_SIZE":                      uint64(4),
		"SHUFFLE_ROUND_COUNT":           uint64(10),
		"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
		"TARGET_COMMITTEE_SIZE":         uint64(4),
		"MAX_COMMITTEES_PER_SLOT":       uint64(4),
		"DOMAIN_PTC_ATTESTER":           []byte{0x0c, 0x00, 0x00, 0x00},
		"DOMAIN_BEACON_ATTESTER":        []byte{0x01, 0x00, 0x00, 0x00},
		"ELECTRA_FORK_EPOCH":            uint64(0),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	// Mix of balances: some high, some low
	validators := make([]*phase0.Validator, 64)
	for i := range validators {
		balance := phase0.Gwei(32_000_000_000)
		if i%2 == 0 {
			balance = phase0.Gwei(2_048_000_000_000) // Max Electra balance
		}

		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey{byte(i)},
			ActivationEpoch:  0,
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: balance,
		}
	}

	genesisBlockHash := phase0.Hash32{0xaa, 0xbb, 0xcc}

	ptcWindow, err := GetGenesisPTCWindow(clConfig, validators, genesisBlockHash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count how often high-balance vs low-balance validators appear in PTC
	highBalanceCount := 0
	lowBalanceCount := 0

	for i := 8; i < len(ptcWindow); i++ {
		for _, idx := range ptcWindow[i] {
			if validators[idx].EffectiveBalance == phase0.Gwei(2_048_000_000_000) {
				highBalanceCount++
			} else {
				lowBalanceCount++
			}
		}
	}

	// High-balance validators (64x more stake) should appear more frequently
	if highBalanceCount <= lowBalanceCount {
		t.Errorf("expected high-balance validators to appear more often: high=%d, low=%d",
			highBalanceCount, lowBalanceCount)
	}
}

func TestGetGenesisPTCWindowDifferentSeeds(t *testing.T) {
	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":               uint64(8),
		"MIN_SEED_LOOKAHEAD":            uint64(1),
		"PTC_SIZE":                      uint64(4),
		"SHUFFLE_ROUND_COUNT":           uint64(10),
		"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
		"TARGET_COMMITTEE_SIZE":         uint64(4),
		"MAX_COMMITTEES_PER_SLOT":       uint64(4),
		"DOMAIN_PTC_ATTESTER":           []byte{0x0c, 0x00, 0x00, 0x00},
		"DOMAIN_BEACON_ATTESTER":        []byte{0x01, 0x00, 0x00, 0x00},
		"ELECTRA_FORK_EPOCH":            uint64(0),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	validators := make([]*phase0.Validator, 64)
	for i := range validators {
		validators[i] = &phase0.Validator{
			PublicKey:        phase0.BLSPubKey{byte(i)},
			ActivationEpoch:  0,
			ExitEpoch:        phase0.Epoch(18446744073709551615),
			EffectiveBalance: phase0.Gwei(32_000_000_000),
		}
	}

	ptcWindow1, err := GetGenesisPTCWindow(clConfig, validators, phase0.Hash32{0x01})
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	ptcWindow2, err := GetGenesisPTCWindow(clConfig, validators, phase0.Hash32{0x02})
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// Different seeds should produce different PTC windows (at least one difference)
	hasDifference := false

	for i := 8; i < len(ptcWindow1); i++ {
		for j := range ptcWindow1[i] {
			if ptcWindow1[i][j] != ptcWindow2[i][j] {
				hasDifference = true

				break
			}
		}

		if hasDifference {
			break
		}
	}

	if !hasDifference {
		t.Error("different genesis block hashes should produce different PTC windows")
	}
}

func TestGetCommitteeCountPerSlot(t *testing.T) {
	tests := []struct {
		name        string
		activeCount uint64
		expected    uint64
	}{
		{
			name:        "small validator set",
			activeCount: 16,
			expected:    1, // 16/8/4 = 0, clamped to 1
		},
		{
			name:        "medium validator set",
			activeCount: 128,
			expected:    4, // 128/8/4 = 4
		},
		{
			name:        "large validator set",
			activeCount: 256,
			expected:    4, // 256/8/4 = 8, clamped to MAX_COMMITTEES_PER_SLOT=4
		},
		{
			name:        "exact boundary",
			activeCount: 32,
			expected:    1, // 32/8/4 = 1
		},
	}

	configValues := map[string]interface{}{
		"SLOTS_PER_EPOCH":         uint64(8),
		"TARGET_COMMITTEE_SIZE":   uint64(4),
		"MAX_COMMITTEES_PER_SLOT": uint64(4),
	}
	clConfig := createTestConfig(t, "minimal", configValues)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCommitteeCountPerSlot(clConfig, tt.activeCount)
			if result != tt.expected {
				t.Errorf("got %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetSlotCommitteeIndices(t *testing.T) {
	// 32 active validators, 8 slots, 1 committee per slot
	// Each slot should get 32/8 = 4 validators
	activeIndices := make([]phase0.ValidatorIndex, 32)
	for i := range activeIndices {
		activeIndices[i] = phase0.ValidatorIndex(i) //nolint:gosec,nolintlint // test index
	}

	seed := phase0.Root{0xab, 0xcd}

	for slot := uint64(0); slot < 8; slot++ {
		indices := getSlotCommitteeIndices(activeIndices, seed, slot, 8, 1, 10)

		if len(indices) != 4 {
			t.Errorf("slot %d: expected 4 indices, got %d", slot, len(indices))
		}

		// All returned indices should be valid
		for _, idx := range indices {
			if uint64(idx) >= 32 {
				t.Errorf("slot %d: invalid index %d", slot, idx)
			}
		}
	}

	// Collect all indices across all slots - should cover all 32 validators
	seen := make(map[phase0.ValidatorIndex]bool, 32)

	for slot := uint64(0); slot < 8; slot++ {
		indices := getSlotCommitteeIndices(activeIndices, seed, slot, 8, 1, 10)
		for _, idx := range indices {
			seen[idx] = true
		}
	}

	if len(seen) != 32 {
		t.Errorf("expected all 32 validators covered, got %d unique", len(seen))
	}
}

func TestComputeBalanceWeightedSelection(t *testing.T) {
	validators := make([]*phase0.Validator, 10)
	for i := range validators {
		validators[i] = &phase0.Validator{
			EffectiveBalance: phase0.Gwei(32_000_000_000),
		}
	}

	candidates := make([]phase0.ValidatorIndex, 10)
	for i := range candidates {
		candidates[i] = phase0.ValidatorIndex(i) //nolint:gosec,nolintlint // test index
	}

	seed := [32]byte{0x01, 0x02, 0x03}
	maxEffBalance := uint64(2_048_000_000_000)

	selected := computeBalanceWeightedSelection(validators, candidates, seed, 4, maxEffBalance)

	if len(selected) != 4 {
		t.Fatalf("expected 4 selected, got %d", len(selected))
	}

	// All selected should be valid indices
	for i, idx := range selected {
		if uint64(idx) >= 10 {
			t.Errorf("pos %d: invalid index %d", i, idx)
		}
	}

	// Determinism check
	selected2 := computeBalanceWeightedSelection(validators, candidates, seed, 4, maxEffBalance)
	for i := range selected {
		if selected[i] != selected2[i] {
			t.Errorf("non-deterministic at pos %d: %d vs %d", i, selected[i], selected2[i])
		}
	}
}
