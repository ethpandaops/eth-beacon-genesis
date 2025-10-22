package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethpandaops/eth-beacon-genesis/buildinfo"
	"github.com/ethpandaops/eth-beacon-genesis/eth1"
	"github.com/ethpandaops/eth-beacon-genesis/leanchain"
	"github.com/ethpandaops/eth-beacon-genesis/leanconfig"
	"github.com/ethpandaops/eth-beacon-genesis/leanvalidators"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func runLeanchain(_ context.Context, cmd *cli.Command) error { //nolint:gocyclo // ignore
	eth1Config := cmd.String(eth1ConfigFlag.Name)
	eth2Config := cmd.String(configFlag.Name)
	massValidatorsFile := cmd.String(massValidatorsFileFlag.Name)
	validatorsFile := cmd.String(validatorsFileFlag.Name)
	stateOutputFile := cmd.String(stateOutputFlag.Name)
	jsonOutputFile := cmd.String(jsonOutputFlag.Name)
	nodesOutputFile := cmd.String(nodesOutputFlag.Name)
	validatorsOutputFile := cmd.String(validatorsOutputFlag.Name)
	configOutputFile := cmd.String(configOutputFlag.Name)
	quiet := cmd.Bool(quietFlag.Name)

	if quiet {
		logrus.SetLevel(logrus.PanicLevel)
	}

	if !quiet {
		logrus.Infof("eth-beacon-genesis version: %s", buildinfo.GetBuildVersion())
	}

	var err error

	var elGenesis *core.Genesis

	if eth1Config != "" {
		elGenesis, err = eth1.LoadEth1GenesisConfig(eth1Config)
		if err != nil {
			return fmt.Errorf("failed to load execution genesis: %w", err)
		}

		logrus.Infof("loaded execution genesis. chainid: %v", elGenesis.Config.ChainID.String())
	}

	var clConfig *leanconfig.Config

	if eth2Config != "" {
		clConfig, err = leanconfig.LoadConfig(eth2Config)
		if err != nil {
			return fmt.Errorf("failed to load consensus config: %w", err)
		}
	} else {
		clConfig, err = leanconfig.DefaultConfig()
		if err != nil {
			return fmt.Errorf("failed to load default consensus config: %w", err)
		}
	}

	logrus.Infof("loaded leanchain config.")

	var clValidators []*leanvalidators.Validator

	if massValidatorsFile != "" {
		vals, err2 := leanvalidators.LoadValidatorsFromMassYaml(massValidatorsFile)
		if err2 != nil {
			return fmt.Errorf("failed to load validators from mass yaml file: %w", err2)
		}

		if len(vals) > 0 {
			clValidators = vals
		}
	}

	if validatorsFile != "" {
		vals, err2 := leanvalidators.LoadValidatorsFromFile(validatorsFile)
		if err2 != nil {
			return fmt.Errorf("failed to load validators from file: %w", err2)
		}

		if len(vals) > 0 {
			clValidators = append(clValidators, vals...)
		}
	}

	if len(clValidators) == 0 {
		return fmt.Errorf("no validators found")
	}

	logrus.Infof("loaded %d validators.", len(clValidators))

	// Generate nodes and validators output if requested
	if nodesOutputFile != "" || validatorsOutputFile != "" {
		err = leanvalidators.GenerateNodeAndValidatorLists(clValidators, nodesOutputFile, validatorsOutputFile)
		if err != nil {
			return fmt.Errorf("failed to generate nodes and validators lists: %w", err)
		}

		if nodesOutputFile != "" {
			logrus.Infof("wrote nodes list to: %s", nodesOutputFile)
		}

		if validatorsOutputFile != "" {
			logrus.Infof("wrote validators list to: %s", validatorsOutputFile)
		}
	}

	builder := leanchain.NewGenesisBuilder(elGenesis, clConfig)
	builder.AddValidators(clValidators)

	genesisState, err := builder.BuildState()
	if err != nil {
		return fmt.Errorf("failed to build genesis: %w", err)
	}

	logrus.Infof("successfully built genesis state.")

	if stateOutputFile != "" {
		sszData, err := builder.Serialize(genesisState, http.ContentTypeSSZ)
		if err != nil {
			return fmt.Errorf("failed to serialize genesis state: %w", err)
		}

		if err := os.WriteFile(stateOutputFile, sszData, 0o644); err != nil { //nolint:gosec // no strict permissions needed
			return fmt.Errorf("failed to write genesis state to SSZ file: %w", err)
		}

		logrus.Infof("serialized genesis state to SSZ file: %s", stateOutputFile)
	}

	if jsonOutputFile != "" {
		jsonData, err := builder.Serialize(genesisState, http.ContentTypeJSON)
		if err != nil {
			return fmt.Errorf("failed to serialize genesis state: %w", err)
		}

		if err := os.WriteFile(jsonOutputFile, jsonData, 0o644); err != nil { //nolint:gosec // no strict permissions needed
			return fmt.Errorf("failed to write genesis state to JSON file: %w", err)
		}

		if !quiet {
			fmt.Printf("serialized genesis state to JSON file: %s\n", jsonOutputFile)
		}
	}

	if stateOutputFile == "" && jsonOutputFile == "" {
		jsonData, err := builder.Serialize(genesisState, http.ContentTypeJSON)
		if err != nil {
			return fmt.Errorf("failed to serialize genesis state: %w", err)
		}

		fmt.Println(string(jsonData))
	}

	// Generate updated config file with VALIDATOR_COUNT if requested
	if configOutputFile != "" && eth2Config != "" {
		err = updateConfigWithValidatorCount(eth2Config, configOutputFile, len(clValidators))
		if err != nil {
			return fmt.Errorf("failed to update config file: %w", err)
		}

		if !quiet {
			logrus.Infof("wrote updated config to: %s", configOutputFile)
		}
	}

	return nil
}

// updateConfigWithValidatorCount reads the input config file, updates VALIDATOR_COUNT using regex to preserve formatting, and writes to output
func updateConfigWithValidatorCount(inputConfigPath, outputConfigPath string, validatorCount int) error {
	// Read the input config file
	configContent, err := os.ReadFile(inputConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read input config file: %w", err)
	}

	// Convert to string for regex processing
	configStr := string(configContent)

	// Define regex pattern to match VALIDATOR_COUNT with various formatting
	// This matches:
	// VALIDATOR_COUNT: 123
	// VALIDATOR_COUNT : 123  (with spaces)
	// VALIDATOR_COUNT: '123' (with quotes)
	// VALIDATOR_COUNT: "123" (with double quotes)
	validatorCountRegex := regexp.MustCompile(`(?m)^(\s*VALIDATOR_COUNT\s*:\s*).*$`)

	newValidatorCountLine := fmt.Sprintf("${1}%d", validatorCount)

	// Check if VALIDATOR_COUNT exists in the config
	if validatorCountRegex.MatchString(configStr) {
		// Update existing VALIDATOR_COUNT
		updatedConfig := validatorCountRegex.ReplaceAllString(configStr, newValidatorCountLine)
		configStr = updatedConfig
	} else {
		// Add VALIDATOR_COUNT at the end of the file
		if !strings.HasSuffix(configStr, "\n") {
			configStr += "\n"
		}

		configStr += fmt.Sprintf("VALIDATOR_COUNT: %d\n", validatorCount)
	}

	// Write the updated config to the output file
	err = os.WriteFile(outputConfigPath, []byte(configStr), 0o644) //nolint:gosec // no strict permissions needed
	if err != nil {
		return fmt.Errorf("failed to write updated config file: %w", err)
	}

	return nil
}
