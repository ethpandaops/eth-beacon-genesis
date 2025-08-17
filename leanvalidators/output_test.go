package leanvalidators

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateNodeAndValidatorLists(t *testing.T) {
	// Create test validators with different ENRs
	validators := []*Validator{
		{ENR: "enr1"},
		{ENR: "enr2"},
		{ENR: "enr3"},
		{ENR: "enr1"}, // Duplicate of first ENR
		{ENR: "enr2"}, // Duplicate of second ENR
		{ENR: "enr1"}, // Another duplicate of first ENR
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

	var validatorLists [][]int
	err = yaml.Unmarshal(validatorsData, &validatorLists)
	if err != nil {
		t.Fatalf("failed to unmarshal validators.yaml: %v", err)
	}

	// Should have 3 lists (one per node)
	if len(validatorLists) != 3 {
		t.Errorf("expected 3 validator lists, got %d", len(validatorLists))
	}

	// Check validator indices
	expectedLists := [][]int{
		{0, 3, 5}, // enr1 appears at indices 0, 3, 5
		{1, 4},    // enr2 appears at indices 1, 4
		{2},       // enr3 appears at index 2
	}

	for i, list := range validatorLists {
		if len(list) != len(expectedLists[i]) {
			t.Errorf("list %d: expected length %d, got %d", i, len(expectedLists[i]), len(list))
			continue
		}
		for j, idx := range list {
			if idx != expectedLists[i][j] {
				t.Errorf("list %d, position %d: expected index %d, got %d", i, j, expectedLists[i][j], idx)
			}
		}
	}
}

func TestGenerateNodeAndValidatorLists_EmptyPaths(t *testing.T) {
	validators := []*Validator{
		{ENR: "enr1"},
		{ENR: "enr2"},
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
		{ENR: "enr1"}, // 0
		{ENR: "enr2"}, // 1
		{ENR: "enr3"}, // 2
		{ENR: "enr1"}, // 3
		{ENR: "enr2"}, // 4
		{ENR: "enr1"}, // 5
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

	var validatorLists [][]int
	err = yaml.Unmarshal(validatorsData, &validatorLists)
	if err != nil {
		t.Fatalf("failed to unmarshal validators.yaml: %v", err)
	}

	// Verify the output matches the expected format from the example
	expectedLists := [][]int{
		{0, 3, 5}, // enr1
		{1, 4},    // enr2
		{2},       // enr3
	}

	for i, list := range validatorLists {
		if len(list) != len(expectedLists[i]) {
			t.Errorf("list %d: expected length %d, got %d", i, len(expectedLists[i]), len(list))
			continue
		}
		for j, idx := range list {
			if idx != expectedLists[i][j] {
				t.Errorf("list %d, position %d: expected index %d, got %d", i, j, expectedLists[i][j], idx)
			}
		}
	}
}