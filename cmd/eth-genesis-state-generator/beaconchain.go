package main

import (
	"context"
	"fmt"
	"os"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/eth-beacon-genesis/beaconchain"
	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/beaconvalidators"
	"github.com/ethpandaops/eth-beacon-genesis/buildinfo"
	"github.com/ethpandaops/eth-beacon-genesis/eth1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func runBeaconchain(ctx context.Context, cmd *cli.Command) error {
	eth1Config := cmd.String(eth1ConfigFlag.Name)
	eth2Config := cmd.String(configFlag.Name)
	mnemonicsFile := cmd.String(mnemonicsFileFlag.Name)
	validatorsFile := cmd.String(validatorsFileFlag.Name)
	shadowForkBlock := cmd.String(shadowForkBlockFlag.Name)
	shadowForkRPC := cmd.String(shadowForkRPCFlag.Name)
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

	clConfig, err := beaconconfig.LoadConfig(eth2Config)
	if err != nil {
		return fmt.Errorf("failed to load consensus config: %w", err)
	}

	logrus.Infof("loaded consensus config. genesis fork version: 0x%x", clConfig.GetBytesDefault("GENESIS_FORK_VERSION", []byte{}))

	var clValidators []*beaconvalidators.Validator

	if mnemonicsFile != "" {
		vals, err2 := beaconvalidators.GenerateValidatorsByMnemonic(mnemonicsFile)
		if err2 != nil {
			return fmt.Errorf("failed to load validators from mnemonics file: %w", err2)
		}

		if len(vals) > 0 {
			clValidators = vals
		}
	}

	if validatorsFile != "" {
		vals, err2 := beaconvalidators.LoadValidatorsFromFile(validatorsFile)
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

	defaultBalance := clConfig.GetUintDefault("MAX_EFFECTIVE_BALANCE", 32_000_000_000)
	totalBalance := uint64(0)
	pubkeyMap := make(map[phase0.BLSPubKey]bool)

	for idx, val := range clValidators {
		if pubkeyMap[val.PublicKey] {
			return fmt.Errorf("duplicate public key in validator set: %s at index %d", val.PublicKey.String(), idx)
		}

		pubkeyMap[val.PublicKey] = true

		if val.Balance != nil {
			totalBalance += *val.Balance
		} else {
			totalBalance += defaultBalance
		}
	}

	logrus.Infof("loaded %d validators. total balance: %d ETH", len(clValidators), totalBalance/1_000_000_000)

	builder := beaconchain.NewGenesisBuilder(elGenesis, clConfig)
	builder.AddValidators(clValidators)

	if shadowForkBlock != "" || shadowForkRPC != "" {
		var gensisBlock *types.Block

		if shadowForkBlock != "" {
			block, err2 := eth1.LoadBlockFromFile(shadowForkBlock)
			if err2 != nil {
				return fmt.Errorf("failed to load shadow fork block from file: %w", err2)
			}

			logrus.Infof("loaded shadow fork block from file. hash: %s", block.Hash().String())

			gensisBlock = block
		} else {
			block, err2 := eth1.GetBlockFromRPC(ctx, shadowForkRPC)
			if err2 != nil {
				return fmt.Errorf("failed to get shadow fork block: %w", err2)
			}

			logrus.Infof("loaded shadow fork block from RPC. hash: %s", block.Hash().String())

			gensisBlock = block
		}

		builder.SetShadowForkBlock(gensisBlock)
	}

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
