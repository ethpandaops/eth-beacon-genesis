package leanvalidators

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// GenerateNodeAndValidatorLists creates nodes.yaml and validators.yaml from a list of validators
func GenerateNodeAndValidatorLists(validators []*Validator, nodesOutputPath, validatorsOutputPath string) error {
	// Track unique ENRs and their indices
	enrToIndex := make(map[string]int)
	nodes := []string{}
	validatorsByNode := make(map[int][]int)

	// Process validators
	for validatorIdx, validator := range validators {
		nodeIdx, exists := enrToIndex[validator.ENR]
		if !exists {
			// New ENR, add to nodes list
			nodeIdx = len(nodes)
			enrToIndex[validator.ENR] = nodeIdx
			nodes = append(nodes, validator.ENR)
			validatorsByNode[nodeIdx] = []int{}
		}
		
		// Add validator index to the node's list
		validatorsByNode[nodeIdx] = append(validatorsByNode[nodeIdx], validatorIdx)
	}

	// Write nodes.yaml if path is provided
	if nodesOutputPath != "" {
		err := writeNodesYAML(nodes, nodesOutputPath)
		if err != nil {
			return fmt.Errorf("failed to write nodes.yaml: %w", err)
		}
	}

	// Write validators.yaml if path is provided
	if validatorsOutputPath != "" {
		// Convert map to slice in order
		validatorLists := make([][]int, len(nodes))
		for nodeIdx := 0; nodeIdx < len(nodes); nodeIdx++ {
			validatorLists[nodeIdx] = validatorsByNode[nodeIdx]
		}
		
		err := writeValidatorsYAML(validatorLists, validatorsOutputPath)
		if err != nil {
			return fmt.Errorf("failed to write validators.yaml: %w", err)
		}
	}

	return nil
}

func writeNodesYAML(nodes []string, outputPath string) error {
	data, err := yaml.Marshal(nodes)
	if err != nil {
		return fmt.Errorf("failed to marshal nodes: %w", err)
	}

	err = os.WriteFile(outputPath, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func writeValidatorsYAML(validatorLists [][]int, outputPath string) error {
	data, err := yaml.Marshal(validatorLists)
	if err != nil {
		return fmt.Errorf("failed to marshal validator lists: %w", err)
	}

	err = os.WriteFile(outputPath, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}