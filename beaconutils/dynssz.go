package beaconutils

import (
	"github.com/ethpandaops/eth-beacon-genesis/config"
	dynssz "github.com/pk910/dynamic-ssz"
)

func GetDynSSZ(cfg *config.Config) *dynssz.DynSsz {
	spec := cfg.GetSpecs()
	dynSsz := dynssz.NewDynSsz(spec)

	return dynSsz
}
