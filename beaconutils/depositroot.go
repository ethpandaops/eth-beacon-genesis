package beaconutils

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	ssz "github.com/ferranbt/fastssz"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/coreutils"
)

func ComputeDepositRoot(cfg *beaconconfig.Config) (phase0.Root, error) {
	// Compute the SSZ hash-tree-root of the empty deposit tree,
	// since that is what we put as eth1_data.deposit_root in the CL genesis state.
	maxDeposits := cfg.GetUintDefault("MAX_DEPOSITS_PER_PAYLOAD", 1<<cfg.GetUintDefault("DEPOSIT_CONTRACT_TREE_DEPTH", 32))

	depositRoot, _ := coreutils.HashWithFastSSZHasher(func(hh *ssz.Hasher) error {
		hh.MerkleizeWithMixin(0, 0, maxDeposits)
		return nil
	})

	return phase0.Root(depositRoot), nil
}
