package main

import (
	"context"
	"fmt"
	"os"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/ethpandaops/eth-beacon-genesis/buildinfo"
	"github.com/ethpandaops/eth-beacon-genesis/eth1"
	"github.com/ethpandaops/eth-beacon-genesis/leanchain"
	"github.com/ethpandaops/eth-beacon-genesis/leanconfig"
	"github.com/ethpandaops/eth-beacon-genesis/leanvalidators"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func runLeanchain(_ context.Context, cmd *cli.Command) error {
	eth1Config := cmd.String(eth1ConfigFlag.Name)
	eth2Config := cmd.String(configFlag.Name)
	validatorsFile := cmd.String(validatorsFileFlag.Name)
	stateOutputFile := cmd.String(stateOutputFlag.Name)
	jsonOutputFile := cmd.String(jsonOutputFlag.Name)
	quiet := cmd.Bool(quietFlag.Name)

	if quiet {
		logrus.SetLevel(logrus.PanicLevel)
	}

	if !quiet {
		logrus.Infof("eth-beacon-genesis version: %s", buildinfo.GetBuildVersion())
	}

	elGenesis, err := eth1.LoadEth1GenesisConfig(eth1Config)
	if err != nil {
		return fmt.Errorf("failed to load execution genesis: %w", err)
	}

	logrus.Infof("loaded execution genesis. chainid: %v", elGenesis.Config.ChainID.String())

	clConfig, err := leanconfig.LoadConfig(eth2Config)
	if err != nil {
		return fmt.Errorf("failed to load consensus config: %w", err)
	}

	logrus.Infof("loaded leanchain config.")

	var clValidators []*leanvalidators.Validator

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

	return nil
}
