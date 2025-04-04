package utils

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	ssz "github.com/ferranbt/fastssz"

	"github.com/ethpandaops/eth-beacon-genesis/config"
)

func ComputeDepositRoot(config *config.Config) (phase0.Root, error) {
	// Compute the SSZ hash-tree-root of the empty deposit tree,
	// since that is what we put as eth1_data.deposit_root in the CL genesis state.
	maxDeposits := config.GetUintDefault("MAX_DEPOSITS_PER_PAYLOAD", 1<<config.GetUintDefault("DEPOSIT_CONTRACT_TREE_DEPTH", 32))

	depositRoot, _ := HashWithFastSSZHasher(func(hh *ssz.Hasher) error {
		hh.MerkleizeWithMixin(0, 0, maxDeposits)
		return nil
	})

	return phase0.Root(depositRoot), nil
}
