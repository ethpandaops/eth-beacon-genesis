package validators

import (
	"crypto/sha256"
	"encoding/binary"

	"golang.org/x/crypto/sha3"
)

const (
	VRFTreeDepth      = 31
	vrfSeedDomain     = "vrf-seed"
	vrfPopulatedDepth = 20
)

func VRFSeed(signingKey []byte) [32]byte {
	return sha256.Sum256(append([]byte(vrfSeedDomain), signingKey...))
}

func VRFLeaf(seed [32]byte, index uint64) [32]byte {
	if index >= 1<<vrfPopulatedDepth {
		return [32]byte{}
	}

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

func vrfLayers(seed [32]byte) [][][32]byte {
	layers := make([][][32]byte, vrfPopulatedDepth+1)
	layers[0] = make([][32]byte, 1<<vrfPopulatedDepth)

	for i := range layers[0] {
		layers[0][i] = VRFLeaf(seed, uint64(i))
	}

	for level := 1; level <= vrfPopulatedDepth; level++ {
		prev := layers[level-1]
		layers[level] = make([][32]byte, len(prev)/2)

		for i := range layers[level] {
			layers[level][i] = hashPair(prev[2*i], prev[2*i+1])
		}
	}

	return layers
}

func zeroHashes() [VRFTreeDepth][32]byte {
	var zeros [VRFTreeDepth][32]byte

	for level := 1; level < VRFTreeDepth; level++ {
		zeros[level] = hashPair(zeros[level-1], zeros[level-1])
	}

	return zeros
}

func VRFRoot(seed [32]byte) [32]byte {
	layers := vrfLayers(seed)
	zeros := zeroHashes()
	node := layers[vrfPopulatedDepth][0]

	for level := vrfPopulatedDepth; level < VRFTreeDepth; level++ {
		node = hashPair(node, zeros[level])
	}

	return node
}

func VRFProof(seed [32]byte, index uint64) [VRFTreeDepth + 1][32]byte {
	var proof [VRFTreeDepth + 1][32]byte

	layers := vrfLayers(seed)
	zeros := zeroHashes()
	first := layers[vrfPopulatedDepth][0]
	node := index

	proof[0] = VRFLeaf(seed, index)

	for level := 0; level < VRFTreeDepth; level++ {
		sibling := node ^ 1

		switch {
		case level < vrfPopulatedDepth && sibling < uint64(len(layers[level])):
			proof[level+1] = layers[level][sibling]
		case level >= vrfPopulatedDepth && sibling == 0:
			proof[level+1] = first
		default:
			proof[level+1] = zeros[level]
		}

		if level >= vrfPopulatedDepth {
			first = hashPair(first, zeros[level])
		}

		node >>= 1
	}

	return proof
}
