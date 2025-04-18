package utils

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	ssz "github.com/ferranbt/fastssz"

	"github.com/ethpandaops/eth-beacon-genesis/config"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
)

func GetGenesisValidators(config *config.Config, validators []*validators.Validator) ([]*phase0.Validator, phase0.Root) {
	// Process activations
	maxEffectiveBalance := phase0.Gwei(config.GetUintDefault("MAX_EFFECTIVE_BALANCE", 32_000_000_000))
	maxEffectiveBalanceElectra := phase0.Gwei(config.GetUintDefault("MAX_EFFECTIVE_BALANCE_ELECTRA", 2_048_000_000_000))
	isElectraActive := false

	if electraActivationEpoch, ok := config.GetUint("ELECTRA_FORK_EPOCH"); ok && electraActivationEpoch == 0 {
		isElectraActive = true
	}

	clValidators := make([]*phase0.Validator, 0, len(validators))

	for i := 0; i < len(validators); i++ {
		val := validators[i]

		if val == nil {
			return nil, phase0.Root{}
		}

		effectiveBalance := phase0.Gwei(0)
		if val.Balance != nil {
			effectiveBalance = phase0.Gwei(*val.Balance)
		} else {
			effectiveBalance = maxEffectiveBalance
		}

		if isElectraActive && val.WithdrawalCredentials[0] == 0x02 {
			// allow electra validators with 0x02 withdrawal credentials to have a higher max effective balance
			if effectiveBalance > maxEffectiveBalanceElectra {
				effectiveBalance = maxEffectiveBalanceElectra
			}
		} else {
			if effectiveBalance > maxEffectiveBalance {
				effectiveBalance = maxEffectiveBalance
			}
		}

		validator := &phase0.Validator{
			PublicKey:                  val.PublicKey,
			WithdrawalCredentials:      val.WithdrawalCredentials,
			EffectiveBalance:           effectiveBalance,
			ActivationEligibilityEpoch: phase0.Epoch(config.GetUintDefault("FAR_FUTURE_EPOCH", 18446744073709551615)),
			ActivationEpoch:            phase0.Epoch(config.GetUintDefault("FAR_FUTURE_EPOCH", 18446744073709551615)),
			ExitEpoch:                  phase0.Epoch(config.GetUintDefault("FAR_FUTURE_EPOCH", 18446744073709551615)),
			WithdrawableEpoch:          phase0.Epoch(config.GetUintDefault("FAR_FUTURE_EPOCH", 18446744073709551615)),
		}

		if effectiveBalance >= maxEffectiveBalance {
			validator.ActivationEligibilityEpoch = phase0.Epoch(0)
			validator.ActivationEpoch = phase0.Epoch(0)
		}

		clValidators = append(clValidators, validator)
	}

	maxValidators := config.GetUintDefault("VALIDATOR_REGISTRY_LIMIT", 1099511627776)
	validatorsRoot, err := HashWithFastSSZHasher(func(hh *ssz.Hasher) error {
		for _, elem := range clValidators {
			if err := elem.HashTreeRootWith(hh); err != nil {
				return err
			}
		}

		hh.MerkleizeWithMixin(0, uint64(len(clValidators)), maxValidators)

		return nil
	})

	if err != nil {
		return nil, phase0.Root{}
	}

	return clValidators, validatorsRoot
}

func GetGenesisBalances(config *config.Config, validators []*validators.Validator) []phase0.Gwei {
	maxEffectiveBalance := phase0.Gwei(config.GetUintDefault("MAX_EFFECTIVE_BALANCE", 32_000_000_000))
	balances := make([]phase0.Gwei, len(validators))

	for i, validator := range validators {
		if validator.Balance != nil {
			balances[i] = phase0.Gwei(*validator.Balance)
		} else {
			balances[i] = maxEffectiveBalance
		}
	}

	return balances
}
