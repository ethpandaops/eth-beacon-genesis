package leanvalidators

import (
	"bufio"
	"fmt"
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

		// Parse line format: "name enr" or just "enr"
		var name, enr string

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			name = strings.TrimSpace(parts[0])
			enr = strings.TrimSpace(parts[1])
		} else {
			name = ""
			enr = strings.TrimSpace(parts[0])
		}

		validatorEntry := &Validator{
			Name: name,
			ENR:  enr,
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

	for i, validator := range validatorList {
		validators = append(validators, &Validator{
			Name: fmt.Sprintf("validator_%d", i),
			ENR:  validator,
		})
	}

	return validators, nil
}
