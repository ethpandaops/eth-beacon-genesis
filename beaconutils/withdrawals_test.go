package beaconutils

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestComputeWithdrawalsRoot(t *testing.T) {
	tests := []struct {
		name          string
		preset        string
		configValues  map[string]interface{}
		withdrawals   types.Withdrawals
		expectedRoot  string
		expectedError bool
	}{
		{
			name:   "empty withdrawals",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_WITHDRAWALS_PER_PAYLOAD": uint64(16),
			},
			withdrawals:  types.Withdrawals{},
			expectedRoot: "792930bbd5baac43bcc798ee49aa8185ef76bb3b44ba62b91d86ae569e4bb535",
		},
		{
			name:   "single withdrawal",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_WITHDRAWALS_PER_PAYLOAD": uint64(16),
			},
			withdrawals: types.Withdrawals{
				&types.Withdrawal{
					Index:     0,
					Validator: 1,
					Address:   common.HexToAddress("0x1234567890123456789012345678901234567890"),
					Amount:    uint64(32000000000),
				},
			},
			expectedRoot: "7f97a0dbe6d693e11d1f21f5602912eb22cc129a84232bc843474ff257f6e537",
		},
		{
			name:   "multiple withdrawals",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_WITHDRAWALS_PER_PAYLOAD": uint64(16),
			},
			withdrawals: types.Withdrawals{
				&types.Withdrawal{
					Index:     0,
					Validator: 1,
					Address:   common.HexToAddress("0x1234567890123456789012345678901234567890"),
					Amount:    uint64(32000000000),
				},
				&types.Withdrawal{
					Index:     1,
					Validator: 2,
					Address:   common.HexToAddress("0x2345678901234567890123456789012345678901"),
					Amount:    uint64(16000000000),
				},
			},
			expectedRoot: "2c1ad2102e56a5c513a3e35557928d83300dfc02b0a733da0bfddc40b2669b50",
		},
		{
			name:   "too many withdrawals",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_WITHDRAWALS_PER_PAYLOAD": uint64(2),
			},
			withdrawals: types.Withdrawals{
				&types.Withdrawal{
					Index:     0,
					Validator: 1,
					Address:   common.Address{},
					Amount:    uint64(1),
				},
				&types.Withdrawal{
					Index:     1,
					Validator: 2,
					Address:   common.Address{},
					Amount:    uint64(1),
				},
				&types.Withdrawal{
					Index:     2,
					Validator: 3,
					Address:   common.Address{},
					Amount:    uint64(1),
				},
			},
			expectedError: true,
		},
		{
			name:   "nil withdrawal",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_WITHDRAWALS_PER_PAYLOAD": uint64(16),
			},
			withdrawals: types.Withdrawals{
				nil,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(t, tt.preset, tt.configValues)

			root, err := ComputeWithdrawalsRoot(tt.withdrawals, cfg)

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedRoot, err := hex.DecodeString(tt.expectedRoot)
			if err != nil {
				t.Fatalf("failed to decode expected root: %v", err)
			}

			if !bytes.Equal(root[:], expectedRoot) {
				t.Errorf("root mismatch: got %x, want %x", root, expectedRoot)
			}
		})
	}
}
