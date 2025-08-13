package beaconutils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
)

// GetGenesisProposers returns the proposer indices for the first 2 epochs
func GetGenesisProposers(clConfig *beaconconfig.Config, validators []*phase0.Validator, genesisBlockHash phase0.Hash32) ([]phase0.ValidatorIndex, error) {
	// Get configuration values
	slotsPerEpoch := clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	totalSlots := slotsPerEpoch * 2 // First 2 epochs

	// Get active validator indices
	activeIndices := []phase0.ValidatorIndex{}

	for i, validator := range validators {
		if validator.ActivationEpoch == 0 { // Active at genesis
			activeIndices = append(activeIndices, phase0.ValidatorIndex(i)) //nolint:gosec // no overflow
		}
	}

	if len(activeIndices) == 0 {
		return nil, fmt.Errorf("no active validators at genesis")
	}

	// Calculate proposers for each slot
	proposers := make([]phase0.ValidatorIndex, totalSlots)

	for slot := uint64(0); slot < totalSlots; slot++ {
		proposers[slot] = computeProposerIndex(clConfig, validators, activeIndices, phase0.Slot(slot), genesisBlockHash)
	}

	return proposers, nil
}

// computeProposerIndex calculates the proposer for a given slot
func computeProposerIndex(clConfig *beaconconfig.Config, validators []*phase0.Validator, activeIndices []phase0.ValidatorIndex, slot phase0.Slot, genesisBlockHash phase0.Hash32) phase0.ValidatorIndex {
	slotsPerEpoch := clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	epoch := phase0.Epoch(uint64(slot) / slotsPerEpoch)

	// Get domain from config
	domainBeaconProposer := clConfig.GetBytesDefault("DOMAIN_BEACON_PROPOSER", []byte{0x00, 0x00, 0x00, 0x00})

	// Get seed for proposer selection using existing seed computation
	seed := computeGenesisSeed(genesisBlockHash, epoch, phase0.DomainType(domainBeaconProposer))

	// Create slot-specific seed
	seedData := make([]byte, 40)
	copy(seedData, seed[:])
	binary.LittleEndian.PutUint64(seedData[32:], uint64(slot))
	slotSeed := sha256.Sum256(seedData)

	// Find proposer using the same algorithm as in temp/duties.go
	shuffleRoundCount := clConfig.GetUintDefault("SHUFFLE_ROUND_COUNT", 90)
	if shuffleRoundCount > 255 {
		shuffleRoundCount = 255
	}

	activeCount := uint64(len(activeIndices))

	// We can safely assume that electra is always activated because the proposer calculation is needed for fulu onwards only
	// use 16-bit random values according to electra specs
	maxEffectiveBalance := clConfig.GetUintDefault("MAX_EFFECTIVE_BALANCE_ELECTRA", 2_048_000_000_000)
	maxRandomValue := uint64(65535) // 2^16 - 1

	for i := uint64(0); ; i++ {
		// Use PermuteIndex for shuffling (same as sync committee selection)
		shuffledIndex := PermuteIndex(
			uint8(shuffleRoundCount), //nolint:gosec // no overflow
			phase0.ValidatorIndex(i%activeCount),
			activeCount,
			phase0.Root(slotSeed),
		)

		// Get the actual validator index
		validatorIndex := activeIndices[shuffledIndex]
		effectiveBalance := validators[validatorIndex].EffectiveBalance

		// Compute random value for this iteration (16-bit)
		buf := make([]byte, 40)
		copy(buf[:32], slotSeed[:])
		binary.LittleEndian.PutUint64(buf[32:], i/16)
		hash := sha256.Sum256(buf)
		offset := (i % 16) * 2
		randomValue := BytesToUint(hash[offset : offset+2])

		// Check if this validator is selected as proposer
		if uint64(effectiveBalance)*maxRandomValue >= maxEffectiveBalance*randomValue {
			return validatorIndex
		}
	}
}
