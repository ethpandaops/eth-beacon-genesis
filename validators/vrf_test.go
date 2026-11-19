package validators

import (
	"encoding/hex"
	"testing"
)

func init() {
	vrfRootDepth = 12
}

func TestVRFNode(t *testing.T) {
	seed := VRFSeed([]byte{1, 2, 3})

	const depth = 18

	layer := make([][32]byte, 1<<depth)
	for i := range layer {
		layer[i] = VRFLeaf(seed, uint64(i))
	}

	for len(layer) > 1 {
		up := make([][32]byte, len(layer)/2)
		for i := range up {
			up[i] = hashPair(layer[2*i], layer[2*i+1])
		}

		layer = up
	}

	if vrfNode(seed, depth, 0) != layer[0] {
		t.Error("node differs from the tree built in full")
	}

	if vrfNode(seed, depth, 1) == layer[0] {
		t.Error("nodes are not distinct")
	}

	seedBytes, _ := hex.DecodeString("543fc5a4d86e79a434e9741770f7ee17c25bdc31be8d61436d7f9836538b03df")

	var pinned [32]byte

	copy(pinned[:], seedBytes)

	if got := vrfNode(pinned, 18, 1); hex.EncodeToString(got[:]) != "a8e932e0dd6ec363635fdb0b30ea60d24cc1a861904c879e236c0e77f9efa562" {
		t.Errorf("node %x differs from the pinned vector", got)
	}
}
