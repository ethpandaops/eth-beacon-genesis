package beaconutils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
)

// GetGenesisPTCWindow computes the PTC (Payload Timeliness Committee) window for the genesis state.
// The window has (2 + MIN_SEED_LOOKAHEAD) * SLOTS_PER_EPOCH entries: the first SLOTS_PER_EPOCH
// entries are zero-filled (empty previous epoch), and the remaining entries contain PTC members
// selected via balance-weighted selection from each slot's beacon committees.
func GetGenesisPTCWindow(clConfig *beaconconfig.Config, validators []*phase0.Validator, genesisBlockHash phase0.Hash32) ([][]phase0.ValidatorIndex, error) {
	slotsPerEpoch := clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	minSeedLookahead := clConfig.GetUintDefault("MIN_SEED_LOOKAHEAD", 1)
	ptcSize := clConfig.GetUintDefault("PTC_SIZE", 512)

	totalSlots := (2 + minSeedLookahead) * slotsPerEpoch

	activeIndices := make([]phase0.ValidatorIndex, 0, len(validators))

	for i, validator := range validators {
		if validator.ActivationEpoch == 0 && validator.ExitEpoch > phase0.Epoch(0) {
			activeIndices = append(activeIndices, phase0.ValidatorIndex(i)) //nolint:G115 // index within validator slice bounds
		}
	}

	if len(activeIndices) == 0 {
		return nil, fmt.Errorf("no active validators at genesis")
	}

	ptcWindow := make([][]phase0.ValidatorIndex, totalSlots)

	// First SLOTS_PER_EPOCH entries are empty (previous epoch placeholder)
	for i := uint64(0); i < slotsPerEpoch; i++ {
		ptcWindow[i] = make([]phase0.ValidatorIndex, ptcSize)
	}

	shuffleRoundCount := clConfig.GetUintDefault("SHUFFLE_ROUND_COUNT", 90)
	if shuffleRoundCount > 255 {
		shuffleRoundCount = 255
	}

	domainPTCAttester := clConfig.GetBytesDefault("DOMAIN_PTC_ATTESTER", []byte{0x0c, 0x00, 0x00, 0x00})
	domainBeaconAttester := clConfig.GetBytesDefault("DOMAIN_BEACON_ATTESTER", []byte{0x01, 0x00, 0x00, 0x00})
	maxEffectiveBalance := clConfig.GetUintDefault("MAX_EFFECTIVE_BALANCE_ELECTRA", 2_048_000_000_000)
	committeesPerSlot := getCommitteeCountPerSlot(clConfig, uint64(len(activeIndices)))

	// Compute PTC for current epoch and lookahead epochs
	for e := uint64(0); e <= minSeedLookahead; e++ {
		epoch := phase0.Epoch(e)

		// Seed for beacon committees (determines which validators are assigned to which slot)
		attesterSeed := computeGenesisSeed(genesisBlockHash, epoch, phase0.DomainType(domainBeaconAttester))

		// Seed for PTC selection
		ptcEpochSeed := computeGenesisSeed(genesisBlockHash, epoch, phase0.DomainType(domainPTCAttester))

		for s := uint64(0); s < slotsPerEpoch; s++ {
			slot := e*slotsPerEpoch + s
			windowIndex := slotsPerEpoch + slot

			// Get concatenated committee indices for this slot
			slotCandidates := getSlotCommitteeIndices(
				activeIndices, attesterSeed, s, slotsPerEpoch, committeesPerSlot,
				uint8(shuffleRoundCount), //nolint:G115 // capped at 255 above
			)

			// Compute slot-specific PTC seed: hash(ptcEpochSeed || uint_to_bytes(slot))
			var seedBuf [40]byte
			copy(seedBuf[:32], ptcEpochSeed[:])
			binary.LittleEndian.PutUint64(seedBuf[32:], slot)
			ptcSlotSeed := sha256.Sum256(seedBuf[:])

			ptcWindow[windowIndex] = computeBalanceWeightedSelection(
				validators, slotCandidates, ptcSlotSeed, ptcSize, maxEffectiveBalance,
			)
		}
	}

	return ptcWindow, nil
}

// getCommitteeCountPerSlot returns the number of beacon committees per slot.
func getCommitteeCountPerSlot(clConfig *beaconconfig.Config, activeCount uint64) uint64 {
	slotsPerEpoch := clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	targetCommitteeSize := clConfig.GetUintDefault("TARGET_COMMITTEE_SIZE", 128)
	maxCommitteesPerSlot := clConfig.GetUintDefault("MAX_COMMITTEES_PER_SLOT", 64)

	count := activeCount / slotsPerEpoch / targetCommitteeSize
	if count > maxCommitteesPerSlot {
		count = maxCommitteesPerSlot
	}

	if count < 1 {
		count = 1
	}

	return count
}

// getSlotCommitteeIndices returns the concatenated validator indices from all beacon
// committees assigned to a given slot within an epoch. The validators are determined by
// shuffling the active set using the attester seed, matching the spec's compute_committee.
func getSlotCommitteeIndices(
	activeIndices []phase0.ValidatorIndex,
	attesterSeed phase0.Root,
	slotInEpoch uint64,
	slotsPerEpoch uint64,
	committeesPerSlot uint64,
	shuffleRounds uint8,
) []phase0.ValidatorIndex {
	activeCount := uint64(len(activeIndices))
	totalCommittees := committeesPerSlot * slotsPerEpoch

	// Contiguous range of shuffled indices covering all committees for this slot
	startIdx := (activeCount * slotInEpoch * committeesPerSlot) / totalCommittees
	endIdx := (activeCount * (slotInEpoch*committeesPerSlot + committeesPerSlot)) / totalCommittees

	indices := make([]phase0.ValidatorIndex, 0, endIdx-startIdx)

	for k := startIdx; k < endIdx; k++ {
		shuffled := PermuteIndex(shuffleRounds, phase0.ValidatorIndex(k), activeCount, attesterSeed)
		indices = append(indices, activeIndices[shuffled])
	}

	return indices
}

// computeBalanceWeightedSelection selects members via balance-weighted random sampling
// without index shuffling (shuffle_indices=False). Uses Electra-style 16-bit random values.
func computeBalanceWeightedSelection(
	validators []*phase0.Validator,
	candidates []phase0.ValidatorIndex,
	seed [32]byte,
	size uint64,
	maxEffectiveBalance uint64,
) []phase0.ValidatorIndex {
	total := uint64(len(candidates))
	if total == 0 {
		return make([]phase0.ValidatorIndex, size)
	}

	selected := make([]phase0.ValidatorIndex, 0, size)
	maxRandomValue := uint64(65535) // 2^16 - 1

	var buf [40]byte

	var h [32]byte

	copy(buf[:32], seed[:])

	for i := uint64(0); uint64(len(selected)) < size; i++ {
		nextIndex := i % total
		candidateIndex := candidates[nextIndex]
		effectiveBalance := uint64(validators[candidateIndex].EffectiveBalance)

		// Generate 16-bit random value; refresh hash source every 16 rounds
		if i%16 == 0 {
			binary.LittleEndian.PutUint64(buf[32:], i/16)
			h = sha256.Sum256(buf[:])
		}

		offset := (i % 16) * 2
		randomValue := BytesToUint(h[offset : offset+2])

		if effectiveBalance*maxRandomValue >= maxEffectiveBalance*randomValue {
			selected = append(selected, candidateIndex)
		}
	}

	return selected
}
