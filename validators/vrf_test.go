package validators

import (
	"testing"
)

func verifyVRFProof(proof *[VRFTreeDepth + 1][32]byte, index uint64, root [32]byte) bool {
	node := proof[0]

	for level := 0; level < VRFTreeDepth; level++ {
		if index>>level&1 == 1 {
			node = hashPair(proof[level+1], node)
		} else {
			node = hashPair(node, proof[level+1])
		}
	}

	return node == root
}

func TestVRFProofs(t *testing.T) {
	seed := VRFSeed([]byte{1, 2, 3})
	root := VRFRoot(seed)

	for _, index := range []uint64{0, 1, 2, 12345, 1<<vrfPopulatedDepth - 1, 1 << vrfPopulatedDepth, 1<<vrfPopulatedDepth + 7, 1<<VRFTreeDepth - 1} {
		proof := VRFProof(seed, index)
		if !verifyVRFProof(&proof, index, root) {
			t.Errorf("proof for leaf %d does not verify", index)
		}

		if index < 1<<vrfPopulatedDepth && verifyVRFProof(&proof, index^1, root) {
			t.Errorf("proof for leaf %d verifies at the wrong index", index)
		}
	}

	if VRFLeaf(seed, 5) == VRFLeaf(seed, 6) {
		t.Error("leaves are not distinct")
	}

	if VRFRoot(VRFSeed([]byte{1, 2, 4})) == root {
		t.Error("roots are not distinct")
	}

	t.Logf("root %x", root)
}
