# eth-beacon-genesis

A tool for generating Ethereum consensus layer (beacon chain) genesis states for development and testing networks.

## Features

- Generate beacon chain genesis states from execution layer genesis and validator configurations
- Support for all forks up to Electra
- Support for validator onboarding via mnemonics or direct key imports
- Configurable genesis parameters
- Output in both SSZ and JSON formats

## Installation

```
go install github.com/ethpandaops/eth-beacon-genesis@latest
```

Or build from source:
```
git clone https://github.com/ethpandaops/eth-beacon-genesis
cd eth-beacon-genesis
make
```
## Usage

Basic usage to generate a beaconchain genesis state:

```
eth-genesis-state-generator beaconchain \
  --eth1-config genesis.json \
  --config config.yaml \
  --mnemonics mnemonics.yaml \
  --state-output genesis.ssz
```

### Command Line Options

- `--eth1-config`: Path to execution layer genesis config (required)
- `--config`: Path to consensus layer config (required) 
- `--mnemonics`: Path to file containing validator mnemonics
- `--additional-validators`: Path to file with additional genesis validators
- `--state-output`: Output path for SSZ genesis state
- `--json-output`: Output path for JSON genesis state
- `--quiet`: Suppress output

### Configuration Files

#### Execution Layer Genesis (genesis.json)
```json
    {
      "config": {
        "chainId": 1337,
        "homesteadBlock": 0
      },
      "nonce": "0x0",
      "timestamp": "0x0",
      "extraData": "0x",
      "gasLimit": "0x1c9c380",
      "difficulty": "0x0",
      "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
      "coinbase": "0x0000000000000000000000000000000000000000",
      "alloc": {}
    }
```

#### Consensus Layer Config (config.yaml)
```yaml
    PRESET_BASE: "mainnet"
    CONFIG_NAME: "devnet"

    # Genesis
    MIN_GENESIS_ACTIVE_VALIDATOR_COUNT: 64
    MIN_GENESIS_TIME: 1606824000
    GENESIS_FORK_VERSION: 0x00000000
    GENESIS_DELAY: 604800

    # Forking
    ALTAIR_FORK_VERSION: 0x01000000
    ALTAIR_FORK_EPOCH: 0
    BELLATRIX_FORK_VERSION: 0x02000000
    BELLATRIX_FORK_EPOCH: 0
    CAPELLA_FORK_VERSION: 0x03000000
    CAPELLA_FORK_EPOCH: 0
    DENEB_FORK_VERSION: 0x04000000
    DENEB_FORK_EPOCH: 0
    ELECTRA_FORK_VERSION: 0x05000000
    ELECTRA_FORK_EPOCH: 0
```

#### Validator Mnemonics File
```yaml
- mnemonic: ""                                             # a 24 word BIP 39 mnemonic
  start: 0                                                 # account index to start from
  count: 100                                               # number of validators to generate
  balance: 32000000000                                     # effective balance
  wd_address: "0x1234567890123456789012345678901234567890" # withdrawal address
  wd_prefix: "0x02"                                        # withdrawal credentials prefix
```

## Development

### Requirements

- Go 1.22+
- Make

### Building

    make build

### Testing

    make test

## License

Apache License 2.0
