package eth1

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// blockJSONWithHash marshals header as a JSON-RPC block object, replacing the
// "hash" field that types.Header.MarshalJSON emits with declaredHash, or
// dropping it entirely when declaredHash is nil.
func blockJSONWithHash(t *testing.T, header *types.Header, declaredHash *common.Hash) json.RawMessage {
	t.Helper()

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("failed to marshal header: %v", err)
	}

	var fields map[string]any
	if err := json.Unmarshal(headerJSON, &fields); err != nil {
		t.Fatalf("failed to unmarshal header into map: %v", err)
	}

	if declaredHash == nil {
		delete(fields, "hash")
	} else {
		fields["hash"] = declaredHash.Hex()
	}

	blockJSON, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("failed to marshal block: %v", err)
	}

	return blockJSON
}

func testHeader() *types.Header {
	return &types.Header{
		UncleHash:   types.EmptyUncleHash,
		Root:        types.EmptyRootHash,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(0),
		GasLimit:    30_000_000,
		Time:        1700000000,
		Extra:       []byte{},
		BaseFee:     big.NewInt(1_000_000_000),
	}
}

func TestParseEthBlockRejectsMismatchedDeclaredHash(t *testing.T) {
	header := testHeader()
	wrongHash := common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	_, err := ParseEthBlock(blockJSONWithHash(t, header, &wrongHash))
	if err == nil {
		t.Fatal("expected an error for a block whose declared hash does not match the recomputed header hash, got nil")
	}

	if !strings.Contains(err.Error(), wrongHash.Hex()) {
		t.Errorf("error should report the declared hash %s, got: %v", wrongHash, err)
	}

	if !strings.Contains(err.Error(), header.Hash().Hex()) {
		t.Errorf("error should report the recomputed hash %s, got: %v", header.Hash(), err)
	}
}

func TestParseEthBlockAcceptsSelfConsistentBlock(t *testing.T) {
	header := testHeader()
	hash := header.Hash()

	for name, declared := range map[string]*common.Hash{
		"declared hash matches": &hash,
		"no declared hash":      nil,
	} {
		t.Run(name, func(t *testing.T) {
			block, err := ParseEthBlock(blockJSONWithHash(t, header, declared))
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if block.Hash() != header.Hash() {
				t.Errorf("expected block hash %s, got %s", header.Hash(), block.Hash())
			}
		})
	}
}
