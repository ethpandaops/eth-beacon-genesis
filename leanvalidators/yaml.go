package leanvalidators

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ShuffleMode string

const (
	ShuffleModeNone       ShuffleMode = "none"
	ShuffleModeRoundRobin ShuffleMode = "roundrobin"
)

type MassValidatorEntry struct {
	ENR   string `yaml:"enr"`
	Count int    `yaml:"count"`
}

type MassValidatorsConfig struct {
	Shuffle    ShuffleMode          `yaml:"shuffle"`
	Validators []MassValidatorEntry `yaml:"validators"`
}

func LoadValidatorsFromMassYaml(validatorsConfigPath string) ([]*Validator, error) {
	validatorsFile, err := os.ReadFile(validatorsConfigPath)
	if err != nil {
		return nil, err
	}

	config := &MassValidatorsConfig{
		Shuffle: ShuffleModeNone, // default
	}

	err = yaml.Unmarshal(validatorsFile, config)
	if err != nil {
		return nil, err
	}

	return expandValidators(config)
}

func expandValidators(config *MassValidatorsConfig) ([]*Validator, error) {
	if len(config.Validators) == 0 {
		return nil, fmt.Errorf("no validators defined in configuration")
	}

	// Calculate total validators
	totalValidators := 0

	for _, entry := range config.Validators {
		if entry.Count < 0 {
			return nil, fmt.Errorf("invalid count %d for ENR %s", entry.Count, entry.ENR)
		}

		totalValidators += entry.Count
	}

	if totalValidators == 0 {
		return nil, fmt.Errorf("no validators to generate (all counts are 0)")
	}

	validators := make([]*Validator, 0, totalValidators)

	switch config.Shuffle {
	case ShuffleModeNone, "":
		// Linear mode: add all validators for each ENR in order
		for _, entry := range config.Validators {
			for i := 0; i < entry.Count; i++ {
				validators = append(validators, &Validator{
					ENR: entry.ENR,
				})
			}
		}

	case ShuffleModeRoundRobin:
		// Round-robin mode: distribute validators evenly
		// Create counters for each ENR
		counters := make([]int, len(config.Validators))
		remaining := totalValidators

		for remaining > 0 {
			for idx, entry := range config.Validators {
				if counters[idx] < entry.Count {
					validators = append(validators, &Validator{
						ENR: entry.ENR,
					})
					counters[idx]++
					remaining--

					if remaining == 0 {
						break
					}
				}
			}
		}

	default:
		return nil, fmt.Errorf("unknown shuffle mode: %s", config.Shuffle)
	}

	return validators, nil
}
