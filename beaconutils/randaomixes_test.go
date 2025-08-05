package beaconutils

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func TestSeedRandomMixes(t *testing.T) {
	tests := []struct {
		name            string
		genesisHash     string
		preset          string
		configValues    map[string]interface{}
		expectedLength  uint64
		expectedAllSame bool
	}{
		{
			name:        "default vector length",
			genesisHash: "4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b",
			preset:      "minimal",
			configValues: map[string]interface{}{
				"EPOCHS_PER_HISTORICAL_VECTOR": uint64(64),
			},
			expectedLength:  64,
			expectedAllSame: true,
		},
		{
			name:        "custom vector length",
			genesisHash: "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			preset:      "minimal",
			configValues: map[string]interface{}{
				"EPOCHS_PER_HISTORICAL_VECTOR": uint64(16),
			},
			expectedLength:  16,
			expectedAllSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genesisHash, err := hex.DecodeString(tt.genesisHash)
			if err != nil {
				t.Fatalf("failed to decode genesis hash: %v", err)
			}

			cfg := createTestConfig(t, tt.preset, tt.configValues)
			randomMixes := SeedRandomMixes(phase0.Hash32(genesisHash), cfg)

			// Check length
			if uint64(len(randomMixes)) != tt.expectedLength {
				t.Errorf("wrong length: got %v, want %v", len(randomMixes), tt.expectedLength)
			}

			// Check all elements are the same as genesis hash
			for i, mix := range randomMixes {
				if !bytes.Equal(mix[:], genesisHash) {
					t.Errorf("mix at index %d differs from genesis hash: got %x, want %x", i, mix, genesisHash)
				}
			}
		})
	}
}
