package beaconutils

import (
	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	dynssz "github.com/pk910/dynamic-ssz"
)

func GetDynSSZ(cfg *beaconconfig.Config) *dynssz.DynSsz {
	spec := cfg.GetSpecs()
	dynSsz := dynssz.NewDynSsz(spec)

	return dynSsz
}
