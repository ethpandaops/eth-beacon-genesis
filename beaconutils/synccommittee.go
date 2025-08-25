package beaconutils

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	blsu "github.com/protolambda/bls12-381-util"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/coreutils"
)

func GetGenesisSyncCommittee(cfg *beaconconfig.Config, validators []*phase0.Validator, randaoMix phase0.Hash32) (*altair.SyncCommittee, error) {
	activeIndices := make([]phase0.ValidatorIndex, 0, len(validators))

	for index, validator := range validators {
		if validator.ActivationEpoch == 0 {
			activeIndices = append(activeIndices, phase0.ValidatorIndex(index)) //nolint:gosec // no overflow
		}
	}

	var committeeIndices []phase0.ValidatorIndex

	if electraActivationEpoch, ok := cfg.GetUint("ELECTRA_FORK_EPOCH"); ok && electraActivationEpoch == 0 {
		committeeIndices = computeGenesisSyncCommitteeIndicesElectra(cfg, activeIndices, validators, randaoMix)
	} else {
		committeeIndices = computeGenesisSyncCommitteeIndices(cfg, activeIndices, validators, randaoMix)
	}

	syncCommittee := &altair.SyncCommittee{
		Pubkeys:         make([]phase0.BLSPubKey, len(committeeIndices)),
		AggregatePubkey: phase0.BLSPubKey{},
	}

	blsPubs := make([]*blsu.Pubkey, 0, len(committeeIndices))

	for i, idx := range committeeIndices {
		var pub blsu.Pubkey
		if err := pub.Deserialize((*[48]byte)(validators[idx].PublicKey[:])); err != nil {
			return nil, err
		}

		syncCommittee.Pubkeys[i] = validators[idx].PublicKey

		blsPubs = append(blsPubs, &pub)
	}

	blsAggregate, err := blsu.AggregatePubkeys(blsPubs)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate sync-committee bls pubkeys")
	}

	syncCommittee.AggregatePubkey = phase0.BLSPubKey(blsAggregate.Serialize())

	return syncCommittee, nil
}

// Return the sequence of sync committee indices (which may include duplicate indices)
// for the next sync committee, given a state at a sync committee period boundary.
//
// Note: Committee can contain duplicate indices for small validator sets (< SYNC_COMMITTEE_SIZE + 128)
func computeGenesisSyncCommitteeIndices(cfg *beaconconfig.Config, active []phase0.ValidatorIndex, validators []*phase0.Validator, randaoMix phase0.Hash32) []phase0.ValidatorIndex {
	syncCommitteeSize := cfg.GetUintDefault("SYNC_COMMITTEE_SIZE", 512)
	shuffleRoundCount := cfg.GetUintDefault("SHUFFLE_ROUND_COUNT", 90)
	maxEffectiveBalance := cfg.GetUintDefault("MAX_EFFECTIVE_BALANCE", 32000000000)
	domainSyncCommittee := cfg.GetBytesDefault("DOMAIN_SYNC_COMMITTEE", []byte{0x07, 0x00, 0x00, 0x00})
	syncCommitteeIndices := make([]phase0.ValidatorIndex, 0, syncCommitteeSize)
	periodSeed := computeGenesisSeed(randaoMix, 0, phase0.DomainType(domainSyncCommittee))

	if len(active) == 0 {
		return syncCommitteeIndices
	}

	var buf [32 + 8]byte

	var h [32]byte

	copy(buf[0:32], periodSeed[:])

	i := phase0.ValidatorIndex(0)

	for uint64(len(syncCommitteeIndices)) < syncCommitteeSize {
		shuffledIndex := PermuteIndex(
			uint8(shuffleRoundCount), //nolint:gosec // no overflow
			i%phase0.ValidatorIndex(len(active)),
			uint64(len(active)),
			periodSeed,
		)
		candidateIndex := active[shuffledIndex]
		validator := validators[candidateIndex]

		effectiveBalance := validator.EffectiveBalance

		// every 32 rounds, create a new source for randomByte
		if i%32 == 0 {
			binary.LittleEndian.PutUint64(buf[32:32+8], uint64(i/32))
			h = sha256.Sum256(buf[:])
		}

		randomByte := h[i%32]
		if effectiveBalance*0xff >= phase0.Gwei(maxEffectiveBalance)*phase0.Gwei(randomByte) {
			syncCommitteeIndices = append(syncCommitteeIndices, candidateIndex)
		}

		i++
	}

	return syncCommitteeIndices
}

func computeGenesisSyncCommitteeIndicesElectra(cfg *beaconconfig.Config, active []phase0.ValidatorIndex, validators []*phase0.Validator, randaoMix phase0.Hash32) []phase0.ValidatorIndex {
	syncCommitteeSize := cfg.GetUintDefault("SYNC_COMMITTEE_SIZE", 512)
	shuffleRoundCount := cfg.GetUintDefault("SHUFFLE_ROUND_COUNT", 90)
	maxEffectiveBalance := cfg.GetUintDefault("MAX_EFFECTIVE_BALANCE", 32000000000)
	domainSyncCommittee := cfg.GetBytesDefault("DOMAIN_SYNC_COMMITTEE", []byte{0x07, 0x00, 0x00, 0x00})
	syncCommitteeIndices := make([]phase0.ValidatorIndex, 0, syncCommitteeSize)
	periodSeed := computeGenesisSeed(randaoMix, 0, phase0.DomainType(domainSyncCommittee))

	if len(active) == 0 {
		return syncCommitteeIndices
	}

	var buf [32 + 8]byte

	var h [32]byte

	copy(buf[0:32], periodSeed[:])

	i := phase0.ValidatorIndex(0)

	for uint64(len(syncCommitteeIndices)) < syncCommitteeSize {
		shuffledIndex := PermuteIndex(
			uint8(shuffleRoundCount), //nolint:gosec // no overflow
			i%phase0.ValidatorIndex(len(active)),
			uint64(len(active)),
			periodSeed,
		)
		candidateIndex := active[shuffledIndex]
		validator := validators[candidateIndex]

		effectiveBalance := validator.EffectiveBalance

		// every 16 rounds, create a new source for randomByte
		if i%16 == 0 {
			binary.LittleEndian.PutUint64(buf[32:32+8], uint64(i/16))
			h = sha256.Sum256(buf[:])
		}

		randomValue := coreutils.BytesToUint(h[(i%16)*2 : (i%16)*2+2])

		if effectiveBalance*0xffff >= phase0.Gwei(maxEffectiveBalance)*phase0.Gwei(randomValue) {
			syncCommitteeIndices = append(syncCommitteeIndices, candidateIndex)
		}

		i++
	}

	return syncCommitteeIndices
}

func computeGenesisSeed(mix phase0.Hash32, epoch phase0.Epoch, domainType phase0.DomainType) phase0.Root {
	data := []byte{}
	data = append(data, domainType[:]...)
	data = append(data, coreutils.UintToBytes(uint64(epoch))...)
	data = append(data, mix[:]...)

	return sha256.Sum256(data)
}

// PermuteIndex shuffles an individual list item without allocating a complete list.
// Returns the index in the would-be shuffled list.
func PermuteIndex(rounds uint8, index phase0.ValidatorIndex, listSize uint64, seed phase0.Root) phase0.ValidatorIndex {
	return innerPermuteIndex(sha256.Sum256, rounds, index, listSize, seed, true)
}

// UnpermuteIndex does the inverse of PermuteIndex,
// it returns the original index when given the same shuffling context parameters and permuted index.
func UnpermuteIndex(rounds uint8, index phase0.ValidatorIndex, listSize uint64, seed phase0.Root) phase0.ValidatorIndex {
	return innerPermuteIndex(sha256.Sum256, rounds, index, listSize, seed, false)
}

const hSeedSize = int8(32)
const hRoundSize = int8(1)
const hPositionWindowSize = int8(4)
const hPivotViewSize = hSeedSize + hRoundSize
const hTotalSize = hSeedSize + hRoundSize + hPositionWindowSize

func innerPermuteIndex(hashFn func([]byte) [32]byte, rounds uint8, input phase0.ValidatorIndex, listSize uint64, seed phase0.Root, dir bool) phase0.ValidatorIndex {
	if rounds == 0 {
		return input
	}

	index := uint64(input)
	buf := make([]byte, hTotalSize)
	r := uint8(0)

	if !dir {
		// Start at last round.
		// Iterating through the rounds in reverse, un-swaps everything, effectively un-shuffling the list.
		r = rounds - 1
	}

	// Seed is always the first 32 bytes of the hash input, we never have to change this part of the buffer.
	copy(buf[:hSeedSize], seed[:])

	for {
		// spec: pivot = bytes_to_int(hash(seed + int_to_bytes1(round))[0:8]) % list_size
		// This is the "int_to_bytes1(round)", appended to the seed.
		buf[hSeedSize] = r
		// Seed is already in place, now just hash the correct part of the buffer, and take a uint64 from it,
		//  and modulo it to get a pivot within range.
		h := hashFn(buf[:hPivotViewSize])
		pivot := binary.LittleEndian.Uint64(h[:8]) % listSize
		// spec: flip = (pivot - index) % list_size
		// Add extra list_size to prevent underflows.
		// "flip" will be the other side of the pair
		flip := (pivot + (listSize - index)) % listSize
		// spec: position = max(index, flip)
		// Why? Don't do double work: we consider every pair only once.
		// (Otherwise we would swap it back in place)
		// Pick the highest index of the pair as position to retrieve randomness with.
		position := index
		if flip > position {
			position = flip
		}
		// spec: source = hash(seed + int_to_bytes1(round) + int_to_bytes4(position // 256))
		// - seed is still in 0:32 (excl., 32 bytes)
		// - round number is still in 32
		// - mix in the position for randomness, except the last byte of it,
		//     which will be used later to select a bit from the resulting hash.
		binary.LittleEndian.PutUint32(buf[hPivotViewSize:], uint32(position>>8)) //nolint:gosec // no overflow
		source := hashFn(buf)
		// spec: byte = source[(position % 256) // 8]
		// Effectively keep the first 5 bits of the byte value of the position,
		//  and use it to retrieve one of the 32 (= 2^5) bytes of the hash.
		byteV := source[(position&0xff)>>3]
		// Using the last 3 bits of the position-byte, determine which bit to get from the hash-byte (8 bits, = 2^3)
		// spec: bit = (byte >> (position % 8)) % 2
		bitV := (byteV >> (position & 0x7)) & 0x1
		// Now that we have our "coin-flip", swap index, or don't.
		// If bitV, flip.
		if bitV == 1 {
			index = flip
		}
		// go forwards?
		if dir {
			// -> shuffle
			r++
			if r == rounds {
				break
			}
		} else {
			if r == 0 {
				break
			}
			// -> un-shuffle
			r--
		}
	}

	return phase0.ValidatorIndex(index)
}
