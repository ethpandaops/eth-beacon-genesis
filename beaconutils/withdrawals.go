package beaconutils

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	ssz "github.com/ferranbt/fastssz"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/coreutils"
)

func ComputeWithdrawalsRoot(withdrawals types.Withdrawals, cfg *beaconconfig.Config) (phase0.Root, error) {
	// Compute the SSZ hash-tree-root of the withdrawals,
	// since that is what we put as withdrawals_root in the CL execution-payload.
	// Not to be confused with the legacy MPT root in the EL block header.
	num := uint64(len(withdrawals))
	maxWithdrawalsPerPayload := cfg.GetUintDefault("MAX_WITHDRAWALS_PER_PAYLOAD", 16)

	if num > maxWithdrawalsPerPayload {
		return phase0.Root{}, fmt.Errorf("withdrawals list is too long")
	}

	clWithdrawals := make([]capella.Withdrawal, len(withdrawals))

	for i, withdrawal := range withdrawals {
		if withdrawal == nil {
			return phase0.Root{}, fmt.Errorf("withdrawal is nil")
		}

		clWithdrawals[i] = capella.Withdrawal{
			Index:          capella.WithdrawalIndex(withdrawal.Index),
			ValidatorIndex: phase0.ValidatorIndex(withdrawal.Validator),
			Address:        bellatrix.ExecutionAddress(withdrawal.Address),
			Amount:         phase0.Gwei(withdrawal.Amount),
		}
	}

	withdrawalsRoot, _ := coreutils.HashWithFastSSZHasher(func(hh *ssz.Hasher) error {
		for _, elem := range clWithdrawals {
			elem.HashTreeRootWith(hh) //nolint:errcheck // no error possible
		}

		hh.MerkleizeWithMixin(0, num, maxWithdrawalsPerPayload)

		return nil
	})

	return phase0.Root(withdrawalsRoot), nil
}
