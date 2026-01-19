package beaconutils

import (
	"github.com/pk910/dynamic-ssz/hasher"
	"github.com/pk910/dynamic-ssz/sszutils"
)

// HashWithFastSSZHasher runs a callback with a Hasher from the default fastssz HasherPool
func HashWithFastSSZHasher(cb func(hh sszutils.HashWalker) error) ([32]byte, error) {
	hh := hasher.DefaultHasherPool.Get()
	if err := cb(hh); err != nil {
		hasher.DefaultHasherPool.Put(hh)
		return [32]byte{}, err
	}

	root, err := hh.HashRoot()
	hasher.DefaultHasherPool.Put(hh)

	return root, err
}
