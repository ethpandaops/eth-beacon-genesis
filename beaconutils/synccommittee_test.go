package beaconutils

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestGetGenesisSyncCommittee(t *testing.T) {
	tests := []struct {
		name                string
		preset              string
		configValues        map[string]interface{}
		validators          []*phase0.Validator
		randaoMix           string
		expectedError       bool
		expectedPubkey      string // First pubkey for verification
		expectedAggregatePk string // Expected aggregate pubkey
	}{
		{
			name:   "single validator",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(18446744073709551615),
			},
			validators: []*phase0.Validator{
				{
					PublicKey:             mustDecodeHexPubkey("b4702b219bcf6691b580aa96814b170713451bcfd75d2f6ebd241df7e4f6b6e30f0ec16c9098242c11c95acade4120ec"),
					WithdrawalCredentials: makeBytes(32, 1),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       0,
				},
			},
			randaoMix:           "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError:       false,
			expectedPubkey:      "b4702b219bcf6691b580aa96814b170713451bcfd75d2f6ebd241df7e4f6b6e30f0ec16c9098242c11c95acade4120ec",
			expectedAggregatePk: "967143d1b28b44b3ff75cba085213bc579dbbe04e35b8d7395f6f4e059f8e44c8af9f12b6174aae223e9c28171eae287",
		},
		{
			name:   "minimal config",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(18446744073709551615), // max uint64 to disable Electra
			},
			validators: []*phase0.Validator{
				{
					PublicKey:             mustDecodeHexPubkey("b4702b219bcf6691b580aa96814b170713451bcfd75d2f6ebd241df7e4f6b6e30f0ec16c9098242c11c95acade4120ec"),
					WithdrawalCredentials: makeBytes(32, 1),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       0,
				},
				{
					PublicKey:             mustDecodeHexPubkey("90588ecdaff043834c21035154c5820d02df74d06535bee41c330871a070a66920c22631574d46bb7e9ce5f890449d7d"),
					WithdrawalCredentials: makeBytes(32, 2),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       0,
				},
				{
					PublicKey:             mustDecodeHexPubkey("a6c0b935ecd925451824d563fa5d5e2dd5c8fe2ae26fed844ee369876896f5f8e764a2cfddc2c86b6e2354249849a829"),
					WithdrawalCredentials: makeBytes(32, 3),
					EffectiveBalance:      16000000000, // Half balance
					ActivationEpoch:       0,
				},
				{
					PublicKey:             mustDecodeHexPubkey("80804dcea8e0a7925083250ee74ec20e1353a9c4d564e98a5cdd9ffee3a3319100cf89b2eb3458718d2baeb6413251f5"),
					WithdrawalCredentials: makeBytes(32, 4),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       1, // Not active at genesis
				},
			},
			randaoMix:           "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError:       false,
			expectedPubkey:      "a6c0b935ecd925451824d563fa5d5e2dd5c8fe2ae26fed844ee369876896f5f8e764a2cfddc2c86b6e2354249849a829",
			expectedAggregatePk: "811a1cc964bcd4314fe43d78791f00f0b57171bf1ca6330b8de4032bb8001b7b5503a92b63728bb8d373867e6dd810d6",
		},
		{
			name:   "electra enabled",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(0),
			},
			validators: []*phase0.Validator{
				{
					PublicKey:             mustDecodeHexPubkey("82cbb3de078c3d305a95b622bc34d1838ba4ba6f95a4e538f11e02b1df4595374fe2069eb1d9ac6c95e83ba1f0dfbe88"),
					WithdrawalCredentials: makeBytes(32, 1),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       0,
				},
				{
					PublicKey:             mustDecodeHexPubkey("90588ecdaff043834c21035154c5820d02df74d06535bee41c330871a070a66920c22631574d46bb7e9ce5f890449d7d"),
					WithdrawalCredentials: makeBytes(32, 2),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       0,
				},
			},
			randaoMix:           "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError:       false,
			expectedPubkey:      "90588ecdaff043834c21035154c5820d02df74d06535bee41c330871a070a66920c22631574d46bb7e9ce5f890449d7d",
			expectedAggregatePk: "83ac2d259c286e5e3b5150fbf876d221f436c8c2e39aaf016290b7498243e2466bd830346cd13e891bad3a84a21ffb46",
		},
		{
			name:   "invalid pubkey",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(0),
			},
			validators: []*phase0.Validator{
				{
					PublicKey:             phase0.BLSPubKey{}, // Invalid empty pubkey
					WithdrawalCredentials: makeBytes(32, 1),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       0,
				},
			},
			randaoMix:     "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError: true,
		},
		{
			name:   "empty validator set",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(18446744073709551615),
			},
			validators:    []*phase0.Validator{},
			randaoMix:     "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError: true,
		},
		{
			name:   "empty validator set electra enabled",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(0),
			},
			validators:    []*phase0.Validator{},
			randaoMix:     "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError: true,
		},
		{
			name:   "no active validators",
			preset: "minimal",
			configValues: map[string]interface{}{
				"SYNC_COMMITTEE_SIZE":   uint64(32),
				"SHUFFLE_ROUND_COUNT":   uint64(10),
				"MAX_EFFECTIVE_BALANCE": uint64(32000000000),
				"DOMAIN_SYNC_COMMITTEE": []byte{0x07, 0x00, 0x00, 0x00},
				"ELECTRA_FORK_EPOCH":    uint64(0),
			},
			validators: []*phase0.Validator{
				{
					PublicKey:             mustDecodeHexPubkey("b4702b219bcf6691b580aa96814b170713451bcfd75d2f6ebd241df7e4f6b6e30f0ec16c9098242c11c95acade4120ec"),
					WithdrawalCredentials: makeBytes(32, 1),
					EffectiveBalance:      32000000000,
					ActivationEpoch:       1, // Not active at genesis
				},
			},
			randaoMix:     "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := createTestConfig(t, tt.preset, tt.configValues)

			randaoMix, err := hex.DecodeString(tt.randaoMix)
			if err != nil {
				t.Fatalf("failed to decode randao mix: %v", err)
			}

			committee, err := GetGenesisSyncCommittee(cfg, tt.validators, phase0.Hash32(randaoMix))

			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check committee size
			expectedSize := cfg.GetUintDefault("SYNC_COMMITTEE_SIZE", 512)
			if uint64(len(committee.Pubkeys)) != expectedSize {
				t.Errorf("wrong committee size: got %v, want %v", len(committee.Pubkeys), expectedSize)
			}

			// Verify first pubkey matches expected
			expectedPubkey, err := hex.DecodeString(tt.expectedPubkey)
			if err != nil {
				t.Fatalf("failed to decode expected pubkey: %v", err)
			}

			if !bytes.Equal(committee.Pubkeys[0][:], expectedPubkey) {
				t.Errorf("first pubkey mismatch: got %x, want %x", committee.Pubkeys[0], expectedPubkey)
			}

			// Verify aggregate pubkey matches expected
			expectedAggregatePk, err := hex.DecodeString(tt.expectedAggregatePk)
			if err != nil {
				t.Fatalf("failed to decode expected aggregate pubkey: %v", err)
			}

			if !bytes.Equal(committee.AggregatePubkey[:], expectedAggregatePk) {
				t.Errorf("aggregate pubkey mismatch: got %x, want %x", committee.AggregatePubkey, expectedAggregatePk)
			}
		})
	}
}

func TestPermuteIndex(t *testing.T) {
	tests := []struct {
		name      string
		rounds    uint8
		index     phase0.ValidatorIndex
		listSize  uint64
		seed      string
		expected  phase0.ValidatorIndex
		unpermute bool
	}{
		{
			name:     "simple permutation",
			rounds:   1,
			index:    0,
			listSize: 32,
			seed:     "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			expected: 0,
		},
		{
			name:     "multiple rounds",
			rounds:   10,
			index:    5,
			listSize: 64,
			seed:     "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected: 27,
		},
		{
			name:      "unpermute",
			rounds:    10,
			index:     31,
			listSize:  64,
			seed:      "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected:  47,
			unpermute: true,
		},
		{
			name:      "no rounds",
			rounds:    0,
			index:     31,
			listSize:  64,
			seed:      "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			expected:  31,
			unpermute: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed, err := hex.DecodeString(tt.seed)
			if err != nil {
				t.Fatalf("failed to decode seed: %v", err)
			}

			var result phase0.ValidatorIndex
			if tt.unpermute {
				result = UnpermuteIndex(tt.rounds, tt.index, tt.listSize, phase0.Root(seed))
			} else {
				result = PermuteIndex(tt.rounds, tt.index, tt.listSize, phase0.Root(seed))
			}

			if result != tt.expected {
				t.Errorf("wrong result: got %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper function to create byte array filled with a specific value
func makeBytes(length int, value byte) []byte {
	b := make([]byte, length)
	for i := range b {
		b[i] = value
	}

	return b
}

// Helper function to decode hex pubkey string to BLSPubKey
func mustDecodeHexPubkey(s string) phase0.BLSPubKey {
	pubkeyBytes, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	var pubkey phase0.BLSPubKey

	copy(pubkey[:], pubkeyBytes)

	return pubkey
}
