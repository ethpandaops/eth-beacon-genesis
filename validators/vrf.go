package validators

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"

	"golang.org/x/crypto/sha3"
)

const (
	VRFTreeDepth     = 31
	vrfSeedDomain    = "vrf-seed"
	vrfParallelDepth = 16
)

func VRFSeed(signingKey []byte) [32]byte {
	return sha256.Sum256(append([]byte(vrfSeedDomain), signingKey...))
}

func VRFLeaf(seed [32]byte, index uint64) [32]byte {
	var buf [40]byte

	copy(buf[:32], seed[:])
	binary.LittleEndian.PutUint64(buf[32:], index)

	return keccak256(buf[:])
}

func hashPair(left, right [32]byte) [32]byte {
	var buf [64]byte

	copy(buf[:32], left[:])
	copy(buf[32:], right[:])

	return keccak256(buf[:])
}

func keccak256(data []byte) [32]byte {
	var out [32]byte

	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	h.Sum(out[:0])

	return out
}

func vrfNode(seed [32]byte, height int, index uint64) [32]byte {
	if height == 0 {
		return VRFLeaf(seed, index)
	}

	if height <= vrfParallelDepth {
		return hashPair(vrfNode(seed, height-1, 2*index), vrfNode(seed, height-1, 2*index+1))
	}

	var (
		wg   sync.WaitGroup
		left [32]byte
	)

	wg.Add(1)

	go func() {
		defer wg.Done()

		left = vrfNode(seed, height-1, 2*index)
	}()

	right := vrfNode(seed, height-1, 2*index+1)

	wg.Wait()

	return hashPair(left, right)
}

var vrfRootDepth = VRFTreeDepth

func VRFRoot(seed [32]byte) [32]byte {
	return vrfNode(seed, vrfRootDepth, 0)
}
