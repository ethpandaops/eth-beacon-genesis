# eth-beacon-genesis

A tool for generating Ethereum consensus layer (beacon chain) genesis states for development and testing networks.

## Features

- Generate beacon chain genesis states from execution layer genesis and validator configurations
- Support for all forks up to Heze
- Support for the consensus-specs `pq` feature fork (`PQ_FORK_EPOCH: 0`), with leanXMSS public keys and VRF roots for genesis validators and builders
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
- `--xmss-pubkey-cache`: Directory caching leanXMSS public keys and VRF roots derived from mnemonics
- `--xmss-pubkey-url`: Base URL to fetch precomputed leanXMSS public keys and VRF roots from
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

#### Post-Quantum Genesis (`pq` fork)

- With `PQ_FORK_EPOCH: 0` the genesis state is the `pq` fork's `BeaconState`: validators and builders are identified by a 32-byte leanXMSS public key, validators carry a VRF root, and there are no sync committees. `HEZE_FORK_EPOCH` must not be scheduled, since `pq` builds on Gloas.
- Validators from mnemonics get their keys from the signing secret key `sk`, the EIP-2333 key at `m/12381/3600/<index>/0/0`, so a client holding the usual validator keystore can derive them. The leanXMSS seed is `sha256("leanxmss-seed" || sk)`. The VRF tree has depth 31 and is hashed with `keccak256`, as `compute_vrf_root` of the specification has it: with `vrf_seed = sha256("vrf-seed" || sk)`, leaf `i` is `keccak256(vrf_seed || uint64_le(i))` for every one of the `2**31` slots.
- Validators from `--additional-validators` carry their keys in two extra columns: `<pubkey>:<withdrawal credentials>:[<balance>]:<xmss pubkey>:<vrf root>`.
- Deriving a key takes a few CPU seconds and its VRF root about an hour of CPU time, since the root covers `2**32` hashes, so anything but a handful of validators needs precomputed keys. `--xmss-pubkey-cache <dir>` stores them in `<dir>/<mnemonic fingerprint>.bin`, where the record of index `i` is the 64 bytes at offset `64*i` (public key, then VRF root) and the fingerprint is the first 8 bytes of `sha256("leanxmss-full-keccak-vrf-keys" || BIP-39 seed)`, in hex. `--xmss-pubkey-url <base url>` fetches the needed range of the same file from a remote location before deriving what is still missing.

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
