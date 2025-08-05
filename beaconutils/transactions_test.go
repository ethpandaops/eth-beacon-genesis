package beaconutils

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestComputeTransactionsRoot(t *testing.T) {
	tests := []struct {
		name          string
		preset        string
		configValues  map[string]interface{}
		transactions  types.Transactions
		expectedRoot  string
		expectedError bool
	}{
		{
			name:   "empty transactions",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_TRANSACTIONS_PER_PAYLOAD": uint64(1048576),
				"MAX_BYTES_PER_TRANSACTION":    uint64(1073741824),
			},
			transactions: types.Transactions{},
			expectedRoot: "7ffe241ea60187fdb0187bfa22de35d1f9bed7ab061d9401fd47e34a54fbede1",
		},
		{
			name:   "single transaction",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_TRANSACTIONS_PER_PAYLOAD": uint64(1048576),
				"MAX_BYTES_PER_TRANSACTION":    uint64(1073741824),
			},
			transactions: types.Transactions{
				types.NewTransaction(
					0, // nonce
					common.HexToAddress("0x1234567890123456789012345678901234567890"), // to
					big.NewInt(1000000000), // value
					21000,                  // gas
					big.NewInt(1000000000), // gasPrice
					[]byte{1, 2, 3, 4},     // data
				),
			},
			expectedRoot: "3c17c2e30ccc5166482251923f3f99a9ba2c35e8ada8810e70944495d1ebd642",
		},
		{
			name:   "multiple transactions",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_TRANSACTIONS_PER_PAYLOAD": uint64(1048576),
				"MAX_BYTES_PER_TRANSACTION":    uint64(1073741824),
			},
			transactions: types.Transactions{
				types.NewTransaction(
					0,
					common.HexToAddress("0x1234567890123456789012345678901234567890"),
					big.NewInt(1000000000),
					21000,
					big.NewInt(1000000000),
					[]byte{1, 2, 3, 4},
				),
				types.NewTransaction(
					1,
					common.HexToAddress("0x2345678901234567890123456789012345678901"),
					big.NewInt(2000000000),
					21000,
					big.NewInt(2000000000),
					[]byte{5, 6, 7, 8},
				),
			},
			expectedRoot: "f7bdb877c2cf675800c04943b4aced3277eadefa592f5e2d377e0c937d706f58",
		},
		{
			name:   "too many transactions",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_TRANSACTIONS_PER_PAYLOAD": uint64(2),
				"MAX_BYTES_PER_TRANSACTION":    uint64(1073741824),
			},
			transactions: types.Transactions{
				types.NewTransaction(0, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
				types.NewTransaction(1, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
				types.NewTransaction(2, common.Address{}, big.NewInt(0), 0, big.NewInt(0), nil),
			},
			expectedError: true,
		},
		{
			name:   "transaction too large",
			preset: "minimal",
			configValues: map[string]interface{}{
				"MAX_TRANSACTIONS_PER_PAYLOAD": uint64(1048576),
				"MAX_BYTES_PER_TRANSACTION":    uint64(10),
			},
			transactions: types.Transactions{
				types.NewTransaction(
					0,
					common.HexToAddress("0x1234567890123456789012345678901234567890"),
					big.NewInt(1000000000),
					21000,
					big.NewInt(1000000000),
					make([]byte, 100), // Large data
				),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(t, tt.preset, tt.configValues)

			root, err := ComputeTransactionsRoot(tt.transactions, cfg)

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
