package leanvalidators

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateNodeAndValidatorLists(t *testing.T) {
	// Create test validators with different ENRs and names
	validators := []*Validator{
		{Name: "validator1", ENR: "enr1"},
		{Name: "validator1", ENR: "enr1"},
		{Name: "validator1", ENR: "enr1"},
		{Name: "validator2", ENR: "enr2"},
		{Name: "validator2", ENR: "enr2"},
		{Name: "validator3", ENR: "enr3"},
	}

	// Create temporary directory for output files
	tmpDir := t.TempDir()
	nodesOutput := filepath.Join(tmpDir, "nodes.yaml")
	validatorsOutput := filepath.Join(tmpDir, "validators.yaml")

	// Generate the files
	err := GenerateNodeAndValidatorLists(validators, nodesOutput, validatorsOutput)
	if err != nil {
		t.Fatalf("failed to generate lists: %v", err)
	}

	// Verify nodes.yaml
	nodesData, err := os.ReadFile(nodesOutput)
	if err != nil {
		t.Fatalf("failed to read nodes.yaml: %v", err)
	}

	var nodes []string

	err = yaml.Unmarshal(nodesData, &nodes)
	if err != nil {
		t.Fatalf("failed to unmarshal nodes.yaml: %v", err)
	}

	// Should have 3 unique ENRs
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}

	// Check nodes are in correct order
	expectedNodes := []string{"enr1", "enr2", "enr3"}
	for i, node := range nodes {
		if node != expectedNodes[i] {
			t.Errorf("node %d: expected %s, got %s", i, expectedNodes[i], node)
		}
	}

	// Verify validators.yaml
	validatorsData, err := os.ReadFile(validatorsOutput)
	if err != nil {
		t.Fatalf("failed to read validators.yaml: %v", err)
	}

	var validatorLists map[string][]int

	err = yaml.Unmarshal(validatorsData, &validatorLists)
	if err != nil {
		t.Fatalf("failed to unmarshal validators.yaml: %v", err)
	}

	// Should have 3 lists (one per node)
	if len(validatorLists) != 3 {
		t.Errorf("expected 3 validator lists, got %d", len(validatorLists))
	}

	// Check validator indices by name
	expectedLists := map[string][]int{
		"validator1": {0, 1, 2}, // validator1 appears at indices 0, 1, 2
		"validator2": {3, 4},    // validator2 appears at indices 3, 4
		"validator3": {5},       // validator3 appears at index 5
	}

	for name, list := range validatorLists {
		if expectedList, exists := expectedLists[name]; exists {
			if len(list) != len(expectedList) {
				t.Errorf("validator %s: expected length %d, got %d", name, len(expectedList), len(list))
				continue
			}

			for j, idx := range list {
				if idx != expectedList[j] {
					t.Errorf("validator %s, position %d: expected index %d, got %d", name, j, expectedList[j], idx)
				}
			}
		} else {
			t.Errorf("unexpected validator name: %s", name)
		}
	}
}

func TestGenerateNodeAndValidatorLists_EmptyPaths(t *testing.T) {
	validators := []*Validator{
		{Name: "test1", ENR: "enr1"},
		{Name: "test2", ENR: "enr2"},
	}

	// Should not error when paths are empty
	err := GenerateNodeAndValidatorLists(validators, "", "")
	if err != nil {
		t.Errorf("unexpected error with empty paths: %v", err)
	}
}

func TestGenerateNodeAndValidatorLists_RoundRobinOrder(t *testing.T) {
	// Test with validators in round-robin order
	validators := []*Validator{
		{Name: "node1_val1", ENR: "enr1"}, // 0
		{Name: "node2_val1", ENR: "enr2"}, // 1
		{Name: "node3_val1", ENR: "enr3"}, // 2
		{Name: "node1_val2", ENR: "enr1"}, // 3
		{Name: "node2_val2", ENR: "enr2"}, // 4
		{Name: "node1_val3", ENR: "enr1"}, // 5
	}

	tmpDir := t.TempDir()
	validatorsOutput := filepath.Join(tmpDir, "validators.yaml")

	err := GenerateNodeAndValidatorLists(validators, "", validatorsOutput)
	if err != nil {
		t.Fatalf("failed to generate lists: %v", err)
	}

	validatorsData, err := os.ReadFile(validatorsOutput)
	if err != nil {
		t.Fatalf("failed to read validators.yaml: %v", err)
	}

	var validatorLists map[string][]int

	err = yaml.Unmarshal(validatorsData, &validatorLists)
	if err != nil {
		t.Fatalf("failed to unmarshal validators.yaml: %v", err)
	}

	// Verify the output matches expected validator names and indices
	expectedLists := map[string][]int{
		"node1_val1": {0}, // first validator
		"node2_val1": {1}, // second validator
		"node3_val1": {2}, // third validator
		"node1_val2": {3}, // fourth validator
		"node2_val2": {4}, // fifth validator
		"node1_val3": {5}, // sixth validator
	}

	for name, list := range validatorLists {
		if expectedList, exists := expectedLists[name]; exists {
			if len(list) != len(expectedList) {
				t.Errorf("validator %s: expected length %d, got %d", name, len(expectedList), len(list))
				continue
			}

			for j, idx := range list {
				if idx != expectedList[j] {
					t.Errorf("validator %s, position %d: expected index %d, got %d", name, j, expectedList[j], idx)
				}
			}
		} else {
			t.Errorf("unexpected validator name: %s", name)
		}
	}
}
