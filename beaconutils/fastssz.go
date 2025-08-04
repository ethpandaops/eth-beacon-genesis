package beaconutils

import ssz "github.com/ferranbt/fastssz"

// HashWithFastSSZHasher runs a callback with a Hasher from the default fastssz HasherPool
func HashWithFastSSZHasher(cb func(hh *ssz.Hasher) error) ([32]byte, error) {
	hh := ssz.DefaultHasherPool.Get()
	if err := cb(hh); err != nil {
		ssz.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}

	root, err := hh.HashRoot()
	ssz.DefaultHasherPool.Put(hh)

	return root, err
}
