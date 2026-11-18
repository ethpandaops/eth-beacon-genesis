package beaconutils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
)

const aggregatorsPerCommittee = 4

func GetGenesisAggregationCommitteeWindow(clConfig *beaconconfig.Config, validators []*phase0.Validator, genesisBlockHash phase0.Hash32) ([][][]phase0.ValidatorIndex, error) {
	slotsPerEpoch := clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	minSeedLookahead := clConfig.GetUintDefault("MIN_SEED_LOOKAHEAD", 1)

	activeIndices := make([]phase0.ValidatorIndex, 0, len(validators))

	for i, validator := range validators {
		if validator.ActivationEpoch == 0 && validator.ExitEpoch > phase0.Epoch(0) {
			activeIndices = append(activeIndices, phase0.ValidatorIndex(i)) //nolint:gosec,nolintlint // index within validator slice bounds
		}
	}

	if len(activeIndices) == 0 {
		return nil, fmt.Errorf("no active validators at genesis")
	}

	shuffleRoundCount := clConfig.GetUintDefault("SHUFFLE_ROUND_COUNT", 90)
	if shuffleRoundCount > 255 {
		shuffleRoundCount = 255
	}

	domainSelectionProof := clConfig.GetBytesDefault("DOMAIN_SELECTION_PROOF", []byte{0x05, 0x00, 0x00, 0x00})
	domainBeaconAttester := clConfig.GetBytesDefault("DOMAIN_BEACON_ATTESTER", []byte{0x01, 0x00, 0x00, 0x00})
	maxEffectiveBalance := clConfig.GetUintDefault("MAX_EFFECTIVE_BALANCE_ELECTRA", 2_048_000_000_000)
	committeesPerSlot := getCommitteeCountPerSlotPQ(clConfig, uint64(len(activeIndices)))

	window := make([][][]phase0.ValidatorIndex, (2+minSeedLookahead)*slotsPerEpoch)

	for i := uint64(0); i < slotsPerEpoch; i++ {
		window[i] = [][]phase0.ValidatorIndex{}
	}

	for e := uint64(0); e <= minSeedLookahead; e++ {
		epoch := phase0.Epoch(e)
		attesterSeed := computeGenesisSeed(genesisBlockHash, epoch, phase0.DomainType(domainBeaconAttester))
		selectionSeed := computeGenesisSeed(genesisBlockHash, epoch, phase0.DomainType(domainSelectionProof))

		for s := uint64(0); s < slotsPerEpoch; s++ {
			slot := e*slotsPerEpoch + s
			committees := make([][]phase0.ValidatorIndex, committeesPerSlot)

			for index := uint64(0); index < committeesPerSlot; index++ {
				candidates := getCommitteeIndices(
					activeIndices, attesterSeed, s*committeesPerSlot+index, committeesPerSlot*slotsPerEpoch,
					uint8(shuffleRoundCount), //nolint:gosec,nolintlint // capped at 255 above
				)

				var seedBuf [48]byte

				copy(seedBuf[:32], selectionSeed[:])
				binary.LittleEndian.PutUint64(seedBuf[32:], slot)
				binary.LittleEndian.PutUint64(seedBuf[40:], index)

				committees[index] = computeAggregationCommittee(validators, candidates, sha256.Sum256(seedBuf[:]), maxEffectiveBalance)
			}

			window[slotsPerEpoch+slot] = committees
		}
	}

	return window, nil
}

func getCommitteeCountPerSlotPQ(clConfig *beaconconfig.Config, activeCount uint64) uint64 {
	slotsPerEpoch := clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	targetCommitteeSize := clConfig.GetUintDefault("TARGET_COMMITTEE_SIZE", 128)
	maxCommitteesPerSlot := clConfig.GetUintDefault("MAX_COMMITTEES_PER_SLOT_PQ", 16)

	return max(1, min(maxCommitteesPerSlot, activeCount/slotsPerEpoch/targetCommitteeSize))
}

func getCommitteeIndices(
	activeIndices []phase0.ValidatorIndex,
	attesterSeed phase0.Root,
	committeeIndex uint64,
	committeeCount uint64,
	shuffleRounds uint8,
) []phase0.ValidatorIndex {
	activeCount := uint64(len(activeIndices))
	startIdx := (activeCount * committeeIndex) / committeeCount
	endIdx := (activeCount * (committeeIndex + 1)) / committeeCount

	indices := make([]phase0.ValidatorIndex, 0, endIdx-startIdx)

	for k := startIdx; k < endIdx; k++ {
		shuffled := PermuteIndex(shuffleRounds, phase0.ValidatorIndex(k), activeCount, attesterSeed)
		indices = append(indices, activeIndices[shuffled])
	}

	return indices
}

func computeAggregationCommittee(
	validators []*phase0.Validator,
	candidates []phase0.ValidatorIndex,
	seed [32]byte,
	maxEffectiveBalance uint64,
) []phase0.ValidatorIndex {
	aggregators := make([]phase0.ValidatorIndex, 0, aggregatorsPerCommittee)

	for len(aggregators) < aggregatorsPerCommittee && len(candidates) > 0 {
		var seedBuf [40]byte

		copy(seedBuf[:32], seed[:])
		binary.LittleEndian.PutUint64(seedBuf[32:], uint64(len(aggregators)))

		aggregator := computeBalanceWeightedSelection(validators, candidates, sha256.Sum256(seedBuf[:]), 1, maxEffectiveBalance)[0]
		aggregators = append(aggregators, aggregator)

		remaining := make([]phase0.ValidatorIndex, 0, len(candidates)-1)

		for _, candidate := range candidates {
			if candidate != aggregator {
				remaining = append(remaining, candidate)
			}
		}

		candidates = remaining
	}

	return aggregators
}
