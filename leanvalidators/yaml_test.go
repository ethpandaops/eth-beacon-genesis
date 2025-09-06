package leanvalidators

import (
	"os"
	"testing"
)

func TestLoadValidatorsFromMassYaml(t *testing.T) {
	// Create a temporary YAML file for testing
	yamlContent := `shuffle: roundrobin
validators:
  # Legacy format
  - name: "lighthouse_validator"
    enr: "enr:-IS4QHCYrYZbAKWCBRlAy5zzaDZXJBGkcnh4MHcBFZntXNFrdvJjX04jRzjzCBOonrkTfj499SZuOh8R33Ls8RRcy5wBgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQPKY0yuDUmstAHYpMa2_oxVtw0RW_QAdpzBQA8yWM0xOIN1ZHCCdl8"
    count: 1

  # New format with privkey
  - name: "prysm_validator"
    privkey: "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
    enrFields:
      ip: "10.0.0.1"
      udp: 30303
      tcp: 30303
      seq: 5
      eth2: "0x00000000"
      custom_field: "test"
    count: 2
`

	tmpFile, err := os.CreateTemp("", "test-mass-validators-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading validators
	validators, err := LoadValidatorsFromMassYaml(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load validators: %v", err)
	}

	// Should have 3 total validators (1 + 2) after expansion
	if len(validators) != 3 {
		t.Errorf("Expected 3 validators, got %d", len(validators))
	}

	// Check that all validators have ENR strings and names
	for i, validator := range validators {
		if validator.ENR == "" {
			t.Errorf("Validator %d has empty ENR", i)
		}
		if validator.Name == "" {
			t.Errorf("Validator %d has empty Name", i)
		}
		t.Logf("Validator %d Name: %s, ENR: %s", i, validator.Name, validator.ENR)
	}

	// Due to round-robin shuffle, validators should be distributed as:
	// lighthouse_validator, prysm_validator, prysm_validator
}

func TestGenerateENRFromPrivKeyAndFields(t *testing.T) {
	privKey := "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	fields := ENRFields{
		IP:  "192.168.1.100",
		TCP: 30303,
		UDP: 30303,
		Seq: 5,
		Custom: map[string]string{
			"eth2":    "0x00000000",
			"attnets": "0xffffffff",
			"test":    "12345",
		},
	}

	enrString, err := generateENRFromPrivKeyAndFields(privKey, fields)
	if err != nil {
		t.Fatalf("Failed to generate ENR: %v", err)
	}

	t.Logf("Generated ENR: %s", enrString)

	// Verify it's a valid ENR string
	if len(enrString) < 10 || enrString[:4] != "enr:" {
		t.Errorf("Generated ENR doesn't look valid: %s", enrString)
	}
}
