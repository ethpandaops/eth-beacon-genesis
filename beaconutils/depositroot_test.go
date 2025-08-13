package beaconutils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"gopkg.in/yaml.v3"
)

func createTestConfig(t *testing.T, preset string, values map[string]interface{}) *beaconconfig.Config {
	t.Helper()

	// Ensure PRESET_BASE is set
	values["PRESET_BASE"] = preset

	// Create temp dir
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	// Convert values to strings for YAML
	yamlValues := make(map[string]string)

	for k, v := range values {
		switch val := v.(type) {
		case uint64:
			yamlValues[k] = fmt.Sprintf("%d", val)
		case []byte:
			yamlValues[k] = fmt.Sprintf("0x%x", val)
		case string:
			yamlValues[k] = val
		default:
			t.Fatalf("unsupported type for config value: %T", v)
		}
	}

	// Write config to file
	yamlData, err := yaml.Marshal(yamlValues)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, yamlData, 0o644); err != nil { //nolint:gosec // test file
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config
	cfg, err := beaconconfig.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	return cfg
}

func TestComputeDepositRoot(t *testing.T) {
	tests := []struct {
		name         string
		preset       string
		configValues map[string]interface{}
		expectedRoot string
	}{
		{
			name:   "default depth (32)",
			preset: "minimal",
			configValues: map[string]interface{}{
				"DEPOSIT_CONTRACT_TREE_DEPTH": uint64(32),
			},
			expectedRoot: "d70a234731285c6804c2a4f56711ddb8c82c99740f207854891028af34e27e5e",
		},
		{
			name:   "custom depth and max deposits",
			preset: "minimal",
			configValues: map[string]interface{}{
				"DEPOSIT_CONTRACT_TREE_DEPTH": uint64(5),
				"MAX_DEPOSITS_PER_PAYLOAD":    uint64(16),
			},
			expectedRoot: "792930bbd5baac43bcc798ee49aa8185ef76bb3b44ba62b91d86ae569e4bb535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := ComputeDepositRoot(createTestConfig(t, tt.preset, tt.configValues))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedRoot, err := hex.DecodeString(tt.expectedRoot)
			if err != nil {
				t.Fatalf("failed to decode expected root: %v", err)
			}

			if len(root) != len(expectedRoot) {
				t.Errorf("root length mismatch: got %d, want %d", len(root), len(expectedRoot))
				return
			}

			if !bytes.Equal(root[:], expectedRoot) {
				t.Errorf("root mismatch: got %x, want %x", root, expectedRoot)
			}
		})
	}
}
