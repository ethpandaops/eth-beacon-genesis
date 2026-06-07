package validators

import (
	"encoding/binary"
	"math/rand/v2"
)

const (
	// minBlockSize and maxBlockSize bound the size of a shuffle block. The
	// validator list is split into contiguous blocks of this many validators
	// and the blocks (not the individual validators) are reordered, preserving
	// each block's internal ordering.
	minBlockSize = 20
	maxBlockSize = 100

	// blockScaleDivisor controls how the block size scales with the total
	// validator count: blockSize = totalCount / blockScaleDivisor, clamped to
	// [minBlockSize, maxBlockSize]. Small sets use minBlockSize, large sets
	// (>= maxBlockSize*blockScaleDivisor validators) use maxBlockSize.
	blockScaleDivisor = 100

	// DefaultShuffleSeed is the seed used for block shuffling when the user does
	// not provide one. A fixed default keeps generation reproducible by default.
	DefaultShuffleSeed uint64 = 0x6765_6e65_7369_73 // "genesis"
)

// SeedFromForkVersion derives a deterministic shuffle seed from the genesis
// fork version. It is used when no explicit --shuffle-seed is provided, so the
// ordering stays reproducible and tied to the network. Falls back to
// DefaultShuffleSeed when the fork version is empty.
func SeedFromForkVersion(forkVersion []byte) uint64 {
	if len(forkVersion) == 0 {
		return DefaultShuffleSeed
	}

	if len(forkVersion) > 8 {
		forkVersion = forkVersion[len(forkVersion)-8:]
	}

	var buf [8]byte

	copy(buf[8-len(forkVersion):], forkVersion)

	return binary.BigEndian.Uint64(buf[:])
}

// blockSizeForCount returns the shuffle block size for a validator set of the
// given size, scaling from minBlockSize (small sets) to maxBlockSize (large
// sets) based on the total count.
func blockSizeForCount(count int) int {
	size := count / blockScaleDivisor

	switch {
	case size < minBlockSize:
		return minBlockSize
	case size > maxBlockSize:
		return maxBlockSize
	default:
		return size
	}
}

// ShuffleValidators reorders the validator list block-wise in place. The list
// is split into contiguous blocks (see blockSizeForCount) and the order of the
// blocks is shuffled using a PRNG seeded with seed, so the same inputs and seed
// always produce the same ordering. Validators within a block keep their
// relative order, which keeps the resulting mapping compact.
func ShuffleValidators(vals []*Validator, seed uint64) {
	n := len(vals)
	if n <= minBlockSize {
		// Nothing meaningful to shuffle: a single block (or less).
		return
	}

	blockSize := blockSizeForCount(n)
	blockCount := (n + blockSize - 1) / blockSize

	order := make([]int, blockCount)
	for i := range order {
		order[i] = i
	}

	rng := rand.New(rand.NewPCG(seed, seed)) //nolint:gosec // deterministic shuffle, not security-sensitive
	rng.Shuffle(blockCount, func(i, j int) {
		order[i], order[j] = order[j], order[i]
	})

	shuffled := make([]*Validator, 0, n)

	for _, block := range order {
		start := block * blockSize
		end := min(start+blockSize, n)
		shuffled = append(shuffled, vals[start:end]...)
	}

	copy(vals, shuffled)
}
