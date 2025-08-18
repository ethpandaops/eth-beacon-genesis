package leanvalidators

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidatorsFromMassYaml(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		expectedCount  int
		expectedError  bool
		validateResult func(t *testing.T, validators []*Validator)
	}{
		{
			name: "linear mode with multiple validators",
			yamlContent: `shuffle: none
validators:
  - enr: enr1
    count: 2
  - enr: enr2
    count: 3`,
			expectedCount: 5,
			validateResult: func(t *testing.T, validators []*Validator) { //nolint:thelper // ignore
				// Should be: enr1, enr1, enr2, enr2, enr2
				expected := []string{"enr1", "enr1", "enr2", "enr2", "enr2"}
				for i, v := range validators {
					if v.ENR != expected[i] {
						t.Errorf("validator %d: expected ENR %s, got %s", i, expected[i], v.ENR)
					}
				}
			},
		},
		{
			name: "round-robin mode",
			yamlContent: `shuffle: roundrobin
validators:
  - enr: enr1
    count: 3
  - enr: enr2
    count: 2`,
			expectedCount: 5,
			validateResult: func(t *testing.T, validators []*Validator) { //nolint:thelper // ignore
				// Should be: enr1, enr2, enr1, enr2, enr1
				expected := []string{"enr1", "enr2", "enr1", "enr2", "enr1"}
				for i, v := range validators {
					if v.ENR != expected[i] {
						t.Errorf("validator %d: expected ENR %s, got %s", i, expected[i], v.ENR)
					}
				}
			},
		},
		{
			name: "default shuffle mode (none)",
			yamlContent: `validators:
  - enr: enr1
    count: 2`,
			expectedCount: 2,
			validateResult: func(t *testing.T, validators []*Validator) { //nolint:thelper // ignore
				for _, v := range validators {
					if v.ENR != "enr1" {
						t.Errorf("expected ENR enr1, got %s", v.ENR)
					}
				}
			},
		},
		{
			name: "zero count validators",
			yamlContent: `validators:
  - enr: enr1
    count: 0
  - enr: enr2
    count: 2`,
			expectedCount: 2,
		},
		{
			name: "all zero counts",
			yamlContent: `validators:
  - enr: enr1
    count: 0`,
			expectedError: true,
		},
		{
			name: "negative count",
			yamlContent: `validators:
  - enr: enr1
    count: -1`,
			expectedError: true,
		},
		{
			name:          "empty validators list",
			yamlContent:   `validators: []`,
			expectedError: true,
		},
		{
			name: "invalid shuffle mode",
			yamlContent: `shuffle: random
validators:
  - enr: enr1
    count: 1`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "validators.yaml")

			err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0o644) //nolint:gosec // no security concern
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Load validators
			validators, err := LoadValidatorsFromMassYaml(tmpFile)

			// Check error expectation
			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got none")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check count
			if len(validators) != tt.expectedCount {
				t.Errorf("expected %d validators, got %d", tt.expectedCount, len(validators))
			}

			// Additional validation
			if tt.validateResult != nil {
				tt.validateResult(t, validators)
			}
		})
	}
}
