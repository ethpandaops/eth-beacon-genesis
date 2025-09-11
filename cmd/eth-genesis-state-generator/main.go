package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/ethpandaops/eth-beacon-genesis/beaconchain"
	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/buildinfo"
	"github.com/ethpandaops/eth-beacon-genesis/eth1"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
)

var (
	eth1ConfigFlag = &cli.StringFlag{
		Name:     "eth1-config",
		Usage:    "Path to execution genesis config (genesis.json)",
		Required: true,
	}
	configFlag = &cli.StringFlag{
		Name:     "config",
		Usage:    "Path to consensus genesis config (config.yaml)",
		Required: true,
	}
	mnemonicsFileFlag = &cli.StringFlag{
		Name:  "mnemonics",
		Usage: "Path to the file containing the mnemonics for genesis validators",
	}
	validatorsFileFlag = &cli.StringFlag{
		Name:  "additional-validators",
		Usage: "Path to the file with a list of additional genesis validators validators",
	}
	shadowForkBlockFlag = &cli.StringFlag{
		Name:  "shadow-fork-block",
		Usage: "Path to the file with a execution block to create a shadow fork from",
	}
	shadowForkRPCFlag = &cli.StringFlag{
		Name:  "shadow-fork-rpc",
		Usage: "Execution RPC URL to fetch the block to create a shadow fork from",
	}
	stateOutputFlag = &cli.StringFlag{
		Name:  "state-output",
		Usage: "Path to the file to write the genesis state to in SSZ format",
	}
	jsonOutputFlag = &cli.StringFlag{
		Name:  "json-output",
		Usage: "Path to the file to write the genesis state to in JSON format",
	}

	quietFlag = &cli.BoolFlag{
		Name:    "quiet",
		Aliases: []string{"q"},
		Usage:   "Suppress output",
	}

	app = &cli.Command{
		Name:  "eth-genesis-state-generator",
		Usage: "Generate ethereum consensus layer genesis states for testnets",
		Commands: []*cli.Command{
			{
				Name:    "beaconchain",
				Usage:   "Generate a beaconchain genesis state",
				Aliases: []string{"bc", "beacon", "devnet"},
				Flags: []cli.Flag{
					eth1ConfigFlag, configFlag, mnemonicsFileFlag, validatorsFileFlag,
					shadowForkBlockFlag, shadowForkRPCFlag, stateOutputFlag, jsonOutputFlag,
					quietFlag,
				},
				Action:    runDevnet,
				UsageText: "eth-beacon-genesis beaconchain [options]",
			},
			{
				Name:  "version",
				Usage: "Print the version of the application",
				Flags: []cli.Flag{},
				Action: func(_ context.Context, _ *cli.Command) error {
					fmt.Printf("eth-beacon-genesis version %s\n", buildinfo.GetBuildVersion())
					return nil
				},
			},
		},
		DefaultCommand: "help",
	}
)

func main() {
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

//nolint:gocyclo // this is a complex function
func runDevnet(ctx context.Context, cmd *cli.Command) error {
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

	var clValidators []*validators.Validator

	if mnemonicsFile != "" {
		vals, err2 := validators.GenerateValidatorsByMnemonic(mnemonicsFile)
		if err2 != nil {
			return fmt.Errorf("failed to load validators from mnemonics file: %w", err2)
		}

		if len(vals) > 0 {
			clValidators = vals
		}
	}

	if validatorsFile != "" {
		vals, err2 := validators.LoadValidatorsFromFile(validatorsFile)
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

	for _, val := range clValidators {
		if val.Balance != nil {
			totalBalance += *val.Balance
		} else {
			totalBalance += defaultBalance
		}
	}

	// check for duplicate public keys
	pubkeyMap := make(map[phase0.BLSPubKey]bool)

	for idx, val := range clValidators {
		if pubkeyMap[val.PublicKey] {
			return fmt.Errorf("duplicate public key in validator set: %s at index %d", val.PublicKey.String(), idx)
		}

		pubkeyMap[val.PublicKey] = true
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
