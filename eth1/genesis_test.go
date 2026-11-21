package eth1

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestGenesisDerivationMatchesFixtures pins the derivation of block 0 from an
// execution genesis config. That derivation is consensus-critical: the state
// root and block hash it produces end up verbatim in the beacon genesis state's
// ExecutionPayloadHeader, and every EL client on the network derives the same
// values independently at startup. A disagreement is otherwise only detected
// when a devnet fails to start.
//
// On the expected values:
//
//   - genesis-empty's state root is the well-known empty Merkle-Patricia trie
//     root, which is fixed by the spec and independent of any implementation.
//     The other values are pinned outputs of the go-ethereum version in go.mod,
//     so they catch a change in the derivation but do not prove it correct.
//   - Cross-check a pinned value against another client with:
//     <client> init --datadir <tmp> <fixture>.json
//     then read block 0 over RPC and compare `stateRoot` and `hash`.
//   - Update a value only alongside a deliberate, understood change in the
//     derivation, never to make a red test green.
func TestGenesisDerivationMatchesFixtures(t *testing.T) {
	check := func(name, file string, wantRoot, wantHash common.Hash) {
		t.Run(name, func(t *testing.T) {
			genesis, err := LoadEth1GenesisConfig(filepath.Join("testdata", file))
			if err != nil {
				t.Fatalf("failed to load %s: %v", file, err)
			}

			block := genesis.ToBlock()

			if block.Root() != wantRoot {
				t.Errorf("state root: expected %s, derived %s", wantRoot, block.Root())
			}

			if block.Hash() != wantHash {
				t.Errorf("block hash: expected %s, derived %s", wantHash, block.Hash())
			}
		})
	}

	check("empty alloc", "genesis-empty.json",
		types.EmptyRootHash,
		common.HexToHash("0x0598047b8adde700d2e815fe0c7436002f7c50ef32447aa4f4bf4c09e1a97789"))

	check("devnet alloc with contract code and storage", "genesis-devnet.json",
		common.HexToHash("0x63068632ffe9270457fc525dcdc6393aa3db76bf19abb1b0965154ee51f1d619"),
		common.HexToHash("0x1b984f36493840bd8e09eda640d0672600356a1ae069e5c1bb91d442d5fe29ca"))
}
