package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ethpandaops/eth-beacon-genesis/buildinfo"
)

var (
	eth1ConfigFlag         *cli.StringFlag
	configFlag             *cli.StringFlag
	mnemonicsFileFlag      *cli.StringFlag
	validatorsFileFlag     *cli.StringFlag
	massValidatorsFileFlag *cli.StringFlag
	shadowForkBlockFlag    *cli.StringFlag
	shadowForkRPCFlag      *cli.StringFlag
	stateOutputFlag        *cli.StringFlag
	jsonOutputFlag         *cli.StringFlag
	nodesOutputFlag        *cli.StringFlag
	validatorsOutputFlag   *cli.StringFlag
	configOutputFlag       *cli.StringFlag
	quietFlag              *cli.BoolFlag
	app                    *cli.Command
)

func init() {
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
	massValidatorsFileFlag = &cli.StringFlag{
		Name:  "mass-validators",
		Usage: "Path to the YAML file containing mass validators configuration with ENRs and counts",
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
	nodesOutputFlag = &cli.StringFlag{
		Name:  "nodes-output",
		Usage: "Path to the file to write the list of nodes (ENRs) to in YAML format",
	}
	validatorsOutputFlag = &cli.StringFlag{
		Name:  "validators-output",
		Usage: "Path to the file to write the validator indices by node to in YAML format",
	}
	configOutputFlag = &cli.StringFlag{
		Name:  "config-output",
		Usage: "Path to write updated consensus config with VALIDATOR_COUNT set",
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
				Action:    runBeaconchain,
				UsageText: "eth-beacon-genesis beaconchain [options]",
			},
			{
				Name:    "leanchain",
				Usage:   "Generate a leanchain genesis state",
				Aliases: []string{"lc", "lean", "leanchain"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "eth1-config",
						Usage:    "Path to execution genesis config (genesis.json)",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "config",
						Usage:    "Path to consensus genesis config (config.yaml)",
						Required: false,
					},
					massValidatorsFileFlag, validatorsFileFlag, stateOutputFlag, jsonOutputFlag,
					nodesOutputFlag, validatorsOutputFlag, configOutputFlag, quietFlag,
				},
				Action:    runLeanchain,
				UsageText: "eth-beacon-genesis leanchain [options]",
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
}

func main() {
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
