package leanchain

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethpandaops/eth-beacon-genesis/leanconfig"
	"github.com/ethpandaops/eth-beacon-genesis/leanvalidators"
)

type NewLeanGenesisBuilderFn func(elGenesis *core.Genesis, clConfig *leanconfig.Config) LeanGenesisBuilder

type LeanGenesisBuilder interface {
	SetShadowForkBlock(block *types.Block)
	AddValidators(validators []*leanvalidators.Validator)
	BuildState() (any, error)
	Serialize(state any, contentType http.ContentType) ([]byte, error)
}

func NewGenesisBuilder(elGenesis *core.Genesis, clConfig *leanconfig.Config) LeanGenesisBuilder {
	return NewPhase0Builder(elGenesis, clConfig)
}
