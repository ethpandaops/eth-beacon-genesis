package leanvalidators

import (
	"bufio"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadValidatorsFromFile(validatorsConfigPath string) ([]*Validator, error) {
	validatorsFile, err := os.Open(validatorsConfigPath)
	if err != nil {
		return nil, err
	}

	defer validatorsFile.Close()

	validators := make([]*Validator, 0)

	scanner := bufio.NewScanner(validatorsFile)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		validatorEntry := &Validator{
			ENR: line,
		}

		validators = append(validators, validatorEntry)
	}

	return validators, nil
}

func LoadValidatorsFromYaml(validatorsConfigPath string) ([]*Validator, error) {
	validatorsFile, err := os.ReadFile(validatorsConfigPath)
	if err != nil {
		return nil, err
	}

	validators := make([]*Validator, 0)

	validatorList := []string{}
	err = yaml.Unmarshal(validatorsFile, &validatorList)
	if err != nil {
		return nil, err
	}

	for _, validator := range validatorList {
		validators = append(validators, &Validator{
			ENR: validator,
		})
	}

	return validators, nil
}
