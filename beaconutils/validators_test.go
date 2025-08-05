package beaconutils

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
)

func TestGetGenesisValidators(t *testing.T) {
	tests := []struct {
		name                string
		preset              string
		configValues        map[string]interface{}
		validators          []*validators.Validator
		expectedRoot        string
		expectedValidators  int
		expectedActivations int
		expectedError       bool
	}{
		{
			name:   "default config",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_EFFECTIVE_BALANCE":    uint64(32_000_000_000),
				"FAR_FUTURE_EPOCH":         uint64(18446744073709551615),
				"VALIDATOR_REGISTRY_LIMIT": uint64(1099511627776),
				"ELECTRA_FORK_EPOCH":       uint64(18446744073709551615), // disabled
			},
			validators: []*validators.Validator{
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 1)),
					WithdrawalCredentials: makeBytes(32, 1),
					Balance:               nil, // should default to MAX_EFFECTIVE_BALANCE
				},
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 2)),
					WithdrawalCredentials: makeBytes(32, 2),
					Balance:               ptr(uint64(16_000_000_000)), // half balance, should not activate
				},
			},
			expectedRoot:        "bb3e018dcc2e297c4c9404a7c17334c7290bdbd11bc0cfbe0dad1410eac00162",
			expectedValidators:  2,
			expectedActivations: 1,
		},
		{
			name:   "electra enabled",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_EFFECTIVE_BALANCE":         uint64(32_000_000_000),
				"MAX_EFFECTIVE_BALANCE_ELECTRA": uint64(2_048_000_000_000),
				"FAR_FUTURE_EPOCH":              uint64(18446744073709551615),
				"VALIDATOR_REGISTRY_LIMIT":      uint64(1099511627776),
				"ELECTRA_FORK_EPOCH":            uint64(0),
			},
			validators: []*validators.Validator{
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 1)),
					WithdrawalCredentials: makeBytes(32, 1),
					Balance:               ptr(uint64(64_000_000_000)), // will be capped at MAX_EFFECTIVE_BALANCE
				},
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 2)),
					WithdrawalCredentials: makeBytes(32, 2),
					Balance:               ptr(uint64(2_049_000_000_000)), // will be allowed up to MAX_EFFECTIVE_BALANCE_ELECTRA
				},
			},
			expectedRoot:        "bd258b3ed92b57d323de641f587466174633109e6147fd3ae106d692c4ceb1fa",
			expectedValidators:  2,
			expectedActivations: 2,
		},
		{
			name:   "invalid withdrawal credentials length",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_EFFECTIVE_BALANCE":    uint64(32_000_000_000),
				"FAR_FUTURE_EPOCH":         uint64(18446744073709551615),
				"VALIDATOR_REGISTRY_LIMIT": uint64(1099511627776),
			},
			validators: []*validators.Validator{
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 1)),
					WithdrawalCredentials: makeBytes(16, 1), // Invalid: only 16 bytes
					Balance:               nil,
				},
			},
			expectedError: true,
		},
		{
			name:   "nil validator",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_EFFECTIVE_BALANCE":    uint64(32_000_000_000),
				"FAR_FUTURE_EPOCH":         uint64(18446744073709551615),
				"VALIDATOR_REGISTRY_LIMIT": uint64(1099511627776),
			},
			validators: []*validators.Validator{
				nil,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(t, tt.preset, tt.configValues)

			vals, root := GetGenesisValidators(cfg, tt.validators)

			if tt.expectedError {
				if vals != nil {
					t.Error("expected error but got validators")
				}

				return
			}

			// Check number of validators
			if len(vals) != tt.expectedValidators {
				t.Errorf("wrong number of validators: got %v, want %v", len(vals), tt.expectedValidators)
			}

			// Check root
			expectedRoot, err := hex.DecodeString(tt.expectedRoot)
			if err != nil {
				t.Fatalf("failed to decode expected root: %v", err)
			}

			if !bytes.Equal(root[:], expectedRoot) {
				t.Errorf("root mismatch: got %x, want %x", root, expectedRoot)
			}

			// Check activations
			activations := 0

			for _, v := range vals {
				if v.ActivationEpoch == 0 {
					activations++
				}
			}

			if activations != tt.expectedActivations {
				t.Errorf("wrong number of activations: got %v, want %v", activations, tt.expectedActivations)
			}
		})
	}
}

func TestGetGenesisBalances(t *testing.T) {
	tests := []struct {
		name          string
		preset        string
		configValues  map[string]interface{}
		validators    []*validators.Validator
		expectedGweis []uint64
		expectedError bool
	}{
		{
			name:   "default balances",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_EFFECTIVE_BALANCE": uint64(32_000_000_000),
			},
			validators: []*validators.Validator{
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 1)),
					WithdrawalCredentials: makeBytes(32, 1),
					Balance:               nil, // should default to MAX_EFFECTIVE_BALANCE
				},
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 2)),
					WithdrawalCredentials: makeBytes(32, 2),
					Balance:               ptr(uint64(16_000_000_000)),
				},
			},
			expectedGweis: []uint64{32_000_000_000, 16_000_000_000},
		},
		{
			name:   "negative balance",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_EFFECTIVE_BALANCE": uint64(32_000_000_000),
			},
			validators: []*validators.Validator{
				{
					PublicKey:             phase0.BLSPubKey(makeBytes(48, 1)),
					WithdrawalCredentials: makeBytes(32, 1),
					Balance:               ptr(uint64(0)), // Zero balance
				},
			},
			expectedGweis: []uint64{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(t, tt.preset, tt.configValues)

			balances := GetGenesisBalances(cfg, tt.validators)

			if len(balances) != len(tt.expectedGweis) {
				t.Fatalf("wrong number of balances: got %v, want %v", len(balances), len(tt.expectedGweis))
			}

			for i, balance := range balances {
				if uint64(balance) != tt.expectedGweis[i] {
					t.Errorf("balance mismatch at index %d: got %v, want %v", i, balance, tt.expectedGweis[i])
				}
			}
		})
	}
}

// Helper function to create pointer to uint64
func ptr(v uint64) *uint64 {
	return &v
}
