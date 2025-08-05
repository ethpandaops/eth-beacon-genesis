package generator

import (
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	hbls "github.com/herumi/bls-eth-go-binary/bls"

	"github.com/ethpandaops/eth-beacon-genesis/config"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
)

type NewGenesisBuilderFn func(elGenesis *core.Genesis, clConfig *config.Config) GenesisBuilder

type GenesisBuilder interface {
	SetShadowForkBlock(block *types.Block)
	AddValidators(validators []*validators.Validator)
	BuildState() (*spec.VersionedBeaconState, error)
	Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error)
}

type ForkConfig struct {
	Version      spec.DataVersion
	EpochField   string
	VersionField string
	BuilderFn    NewGenesisBuilderFn
}

var ForkConfigs = []ForkConfig{
	{
		Version:      spec.DataVersionPhase0,
		EpochField:   "",
		VersionField: "GENESIS_FORK_VERSION",
		BuilderFn:    NewPhase0Builder,
	},
	{
		Version:      spec.DataVersionAltair,
		EpochField:   "ALTAIR_FORK_EPOCH",
		VersionField: "ALTAIR_FORK_VERSION",
		BuilderFn:    NewAltairBuilder,
	},
	{
		Version:      spec.DataVersionBellatrix,
		EpochField:   "BELLATRIX_FORK_EPOCH",
		VersionField: "BELLATRIX_FORK_VERSION",
		BuilderFn:    NewBellatrixBuilder,
	},
	{
		Version:      spec.DataVersionCapella,
		EpochField:   "CAPELLA_FORK_EPOCH",
		VersionField: "CAPELLA_FORK_VERSION",
		BuilderFn:    NewCapellaBuilder,
	},
	{
		Version:      spec.DataVersionDeneb,
		EpochField:   "DENEB_FORK_EPOCH",
		VersionField: "DENEB_FORK_VERSION",
		BuilderFn:    NewDenebBuilder,
	},
	{
		Version:      spec.DataVersionElectra,
		EpochField:   "ELECTRA_FORK_EPOCH",
		VersionField: "ELECTRA_FORK_VERSION",
		BuilderFn:    NewElectraBuilder,
	},
	{
		Version:      spec.DataVersionFulu,
		EpochField:   "FULU_FORK_EPOCH",
		VersionField: "FULU_FORK_VERSION",
		BuilderFn:    NewFuluBuilder,
	},
}

func init() {
	//nolint:errcheck // ignore
	hbls.Init(hbls.BLS12_381)
	//nolint:errcheck // ignore
	hbls.SetETHmode(hbls.EthModeLatest)
}

func GetGenesisForkVersion(clConfig *config.Config) spec.DataVersion {
	for i := len(ForkConfigs) - 1; i >= 1; i-- {
		if epoch, found := clConfig.GetUint(ForkConfigs[i].EpochField); found && epoch == 0 {
			return ForkConfigs[i].Version
		}
	}

	return spec.DataVersionPhase0
}

func GetForkConfig(version spec.DataVersion) *ForkConfig {
	for _, forkConfig := range ForkConfigs {
		if forkConfig.Version == version {
			return &forkConfig
		}
	}

	return nil
}

func GetStateForkConfig(version spec.DataVersion, cfg *config.Config) *phase0.Fork {
	thisForkConfig := GetForkConfig(version)

	var prevForkConfig *ForkConfig

	if version == spec.DataVersionPhase0 {
		prevForkConfig = thisForkConfig
	} else {
		prevForkConfig = GetForkConfig(version - 1)
	}

	thisForkVersion, _ := cfg.GetBytes(thisForkConfig.VersionField)
	prevForkVersion, _ := cfg.GetBytes(prevForkConfig.VersionField)

	return &phase0.Fork{
		CurrentVersion:  phase0.Version(thisForkVersion),
		PreviousVersion: phase0.Version(prevForkVersion),
		Epoch:           0,
	}
}

func NewGenesisBuilder(elGenesis *core.Genesis, clConfig *config.Config) GenesisBuilder {
	forkVersion := GetGenesisForkVersion(clConfig)
	forkConfig := GetForkConfig(forkVersion)

	if forkConfig == nil {
		return nil
	}

	return forkConfig.BuilderFn(elGenesis, clConfig)
}
