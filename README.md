# eth-beacon-genesis

A tool for generating Ethereum consensus layer (beacon chain) genesis states for development and testing networks.

## Features

- Generate beacon chain genesis states from execution layer genesis and validator configurations
- Support for all forks up to Heze
- Support for the EIP-8450 post-quantum key registry on top of Heze, with leanXMSS keys for genesis validators and builders
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
- `--shuffle-validators`: Shuffle the validator set block-wise to add variance to the validator ordering
- `--xmss-pubkey-cache`: Directory caching leanXMSS public keys derived from mnemonics
- `--xmss-pubkey-url`: Base URL to fetch precomputed leanXMSS public keys from
- `--xmss-validator-pubkeys`: Use leanXMSS public keys as the validators' public keys
- `--shuffle-seed`: Seed for the block-wise validator shuffle (defaults to the genesis fork version; only used with `--shuffle-validators`)
- `--validators-mapping-output`: Output path for the validator mapping (state index ranges to source key ranges) in YAML format
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
    FULU_FORK_VERSION: 0x06000000
    FULU_FORK_EPOCH: 0
    GLOAS_FORK_VERSION: 0x07000000
    GLOAS_FORK_EPOCH: 0
    HEZE_FORK_VERSION: 0x08000000
    HEZE_FORK_EPOCH: 0
    EIP8450_FORK_VERSION: 0xe8450000
    EIP8450_FORK_EPOCH: 0
```

#### Validator Mnemonics File
```yaml
- mnemonic: ""                                             # a 24 word BIP 39 mnemonic
  name: ""                                                 # optional source name used in the validator mapping (defaults to mnemonic-<index>)
  start: 0                                                 # account index to start from
  count: 100                                               # number of validators to generate
  balance: 32000000000                                     # effective balance
  wd_address: "0x1234567890123456789012345678901234567890" # withdrawal address
  wd_prefix: "0x02"                                        # withdrawal credentials prefix
  status: 0                                                # validator status: 0=active, 1=slashed, 2=exited
```

#### Post-Quantum Keys (EIP-8450)

When `EIP8450_FORK_EPOCH` is `0`, the genesis state carries the EIP-8450 key registry (`post_quantum_key_material`, `post_quantum_key_change_epochs` and their builder counterparts).

- Validators from mnemonics get a leanXMSS public key. Its 32-byte seed is `sha256("leanxmss-seed" || signing secret key)`, where the signing secret key is the validator's EIP-2333 key at `m/12381/3600/<index>/0/0`, so a client holding the usual validator keystore can derive it. The VRF root is left zero.
- `--xmss-validator-pubkeys` replaces each validator's BLS public key with its leanXMSS public key, zero-padded to 48 bytes, and leaves the sync committee aggregate public keys zero. Use it for networks whose clients sign with leanXMSS instead of BLS.
- Validators from `--additional-validators` can carry keys in two extra columns: `<pubkey>:<withdrawal credentials>:[<balance>]:<xmss pubkey>[:<vrf root>]`. Validators without them get an unset registry entry.
- Deriving a key takes a few CPU seconds. `--xmss-pubkey-cache <dir>` stores derived keys in `<dir>/<mnemonic fingerprint>.bin`, where the key of index `i` is the 32 bytes at offset `32*i` and the fingerprint is the first 8 bytes of `sha256("leanxmss-seed" || BIP-39 seed)`, in hex. `--xmss-pubkey-url <base url>` fetches the needed range of the same file from a remote location before deriving what is still missing.

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
