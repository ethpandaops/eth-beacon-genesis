package leanchain

import (
	"encoding/json"
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/go-lean-types/specs/phase0"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/eth-beacon-genesis/leanconfig"
	"github.com/ethpandaops/eth-beacon-genesis/leanutils"
	"github.com/ethpandaops/eth-beacon-genesis/leanvalidators"
	dynssz "github.com/pk910/dynamic-ssz"
)

type phase0Builder struct {
	elGenesis       *core.Genesis
	clConfig        *leanconfig.Config
	dynSsz          *dynssz.DynSsz
	shadowForkBlock *types.Block
	validators      []*leanvalidators.Validator
}

func NewPhase0Builder(elGenesis *core.Genesis, clConfig *leanconfig.Config) LeanGenesisBuilder {
	return &phase0Builder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
		dynSsz:    leanutils.GetDynSSZ(clConfig),
	}
}

func (b *phase0Builder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *phase0Builder) AddValidators(val []*leanvalidators.Validator) {
	b.validators = append(b.validators, val...)
}

func (b *phase0Builder) BuildState() (any, error) {
	genesisState := &phase0.State{
		Config: phase0.Config{
			NumValidators: uint64(len(b.validators)),
		},
		LatestJustified:          phase0.Checkpoint{},
		LatestFinalized:          phase0.Checkpoint{},
		HistoricalBlockHashes:    []phase0.Root{},
		JustifiedSlots:           []bool{},
		JustificationsRoots:      []phase0.Root{},
		JustificationsValidators: []byte{},
	}

	logrus.Infof("genesis version: phase0")

	return genesisState, nil
}

func (b *phase0Builder) Serialize(state any, contentType http.ContentType) ([]byte, error) {
	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(state)
	case http.ContentTypeJSON:
		return json.Marshal(state)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
