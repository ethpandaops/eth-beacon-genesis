package leanutils

import (
	"github.com/ethpandaops/eth-beacon-genesis/leanconfig"
	dynssz "github.com/pk910/dynamic-ssz"
)

func GetDynSSZ(cfg *leanconfig.Config) *dynssz.DynSsz {
	spec := cfg.GetSpecs()
	dynSsz := dynssz.NewDynSsz(spec)

	return dynSsz
}
