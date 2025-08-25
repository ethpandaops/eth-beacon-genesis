package beaconutils

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	ssz "github.com/ferranbt/fastssz"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/coreutils"
)

func ComputeTransactionsRoot(transactions types.Transactions, cfg *beaconconfig.Config) (phase0.Root, error) {
	// Compute the SSZ hash-tree-root of the transactions,
	// since that is what we put as transactions_root in the CL execution-payload.
	// Not to be confused with the legacy MPT root in the EL block header.
	num := uint64(len(transactions))
	maxTransactionsPerPayload := cfg.GetUintDefault("MAX_TRANSACTIONS_PER_PAYLOAD", 1048576)

	if num > maxTransactionsPerPayload {
		return phase0.Root{}, fmt.Errorf("transactions list is too long")
	}

	clTransactions := make([]bellatrix.Transaction, len(transactions))

	for i, tx := range transactions {
		opaqueTx, err := tx.MarshalBinary()
		if err != nil {
			return phase0.Root{}, fmt.Errorf("failed to encode tx %d: %w", i, err)
		}

		clTransactions[i] = opaqueTx
	}

	maxBytesPerTx := cfg.GetUintDefault("MAX_BYTES_PER_TRANSACTION", 1073741824)

	transactionsRoot, err := coreutils.HashWithFastSSZHasher(func(hh *ssz.Hasher) error {
		for i, elem := range clTransactions {
			elemIndx := hh.Index()
			byteLen := uint64(len(elem))

			if byteLen > maxBytesPerTx {
				return fmt.Errorf("transaction %d is too long", i)
			}

			hh.AppendBytes32(elem)
			hh.MerkleizeWithMixin(elemIndx, byteLen, (maxBytesPerTx+31)/32)
		}

		hh.MerkleizeWithMixin(0, num, maxTransactionsPerPayload)

		return nil
	})
	if err != nil {
		return phase0.Root{}, err
	}

	return phase0.Root(transactionsRoot), nil
}
