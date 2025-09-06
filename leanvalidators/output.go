package leanvalidators

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// GenerateNodeAndValidatorLists creates nodes.yaml and validators.yaml from a list of validators
func GenerateNodeAndValidatorLists(validators []*Validator, nodesOutputPath, validatorsOutputPath string) error {
	// Track unique ENRs for nodes.yaml
	uniqueENRs := make(map[string]bool)
	nodes := []string{}

	// Track validator indices by validator name for validators.yaml
	validatorIndicesByName := make(map[string][]int)

	// Process validators
	for validatorIdx, validator := range validators {
		// Add unique ENRs to nodes list
		if !uniqueENRs[validator.ENR] {
			uniqueENRs[validator.ENR] = true
			nodes = append(nodes, validator.ENR)
		}

		// Group validator indices by name
		validatorName := validator.Name
		if validatorName == "" {
			validatorName = fmt.Sprintf("validator_%d", validatorIdx)
		}

		validatorIndicesByName[validatorName] = append(validatorIndicesByName[validatorName], validatorIdx)
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
		err := writeValidatorsYAML(validatorIndicesByName, validatorsOutputPath)
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

	err = os.WriteFile(outputPath, data, 0o644) //nolint:gosec // no security concern
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func writeValidatorsYAML(validatorIndicesByName map[string][]int, outputPath string) error {
	data, err := yaml.Marshal(validatorIndicesByName)
	if err != nil {
		return fmt.Errorf("failed to marshal validator lists: %w", err)
	}

	err = os.WriteFile(outputPath, data, 0o644) //nolint:gosec // no security concern
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
