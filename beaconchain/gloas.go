package beaconchain

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/beaconutils"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
	dynssz "github.com/pk910/dynamic-ssz"
)

type gloasBuilder struct {
	elGenesis       *core.Genesis
	clConfig        *beaconconfig.Config
	dynSsz          *dynssz.DynSsz
	shadowForkBlock *types.Block
	validators      []*validators.Validator
}

func NewGloasBuilder(elGenesis *core.Genesis, clConfig *beaconconfig.Config) BeaconGenesisBuilder {
	return &gloasBuilder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
		dynSsz:    beaconutils.GetDynSSZ(clConfig),
	}
}

func (b *gloasBuilder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *gloasBuilder) AddValidators(val []*validators.Validator) {
	b.validators = append(b.validators, val...)
}

func (b *gloasBuilder) BuildState() (*spec.VersionedBeaconState, error) {
	genesisBlock := b.shadowForkBlock
	if genesisBlock == nil {
		genesisBlock = b.elGenesis.ToBlock()
	}

	genesisBlockHash := genesisBlock.Hash()

	extra := genesisBlock.Extra()
	if len(extra) > 32 {
		return nil, fmt.Errorf("extra data is %d bytes, max is %d", len(extra), 32)
	}

	depositRoot, err := beaconutils.ComputeDepositRoot(b.clConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to compute deposit root: %w", err)
	}

	syncCommitteeSize := b.clConfig.GetUintDefault("SYNC_COMMITTEE_SIZE", 512)
	syncCommitteeMaskBytes := syncCommitteeSize / 8

	if syncCommitteeSize%8 != 0 {
		syncCommitteeMaskBytes++
	}

	genesisBlockBody := &electra.BeaconBlockBody{
		ETH1Data: &phase0.ETH1Data{
			BlockHash: make([]byte, 32),
		},
		SyncAggregate: &altair.SyncAggregate{
			SyncCommitteeBits: make([]byte, syncCommitteeMaskBytes),
		},
		ExecutionPayload: &deneb.ExecutionPayload{
			BaseFeePerGas: uint256.NewInt(0),
		},
		ExecutionRequests: &electra.ExecutionRequests{},
	}

	genesisBlockBodyRoot, err := b.dynSsz.HashTreeRoot(genesisBlockBody)
	if err != nil {
		return nil, fmt.Errorf("failed to compute genesis block body root: %w", err)
	}

	builders, validators := beaconutils.SeparateBuildersFromValidators(b.validators)
	clValidators, validatorsRoot := beaconutils.GetGenesisValidators(b.clConfig, validators)
	clBuilders := beaconutils.GetGenesisBuilders(b.clConfig, builders)

	syncCommittee, err := beaconutils.GetGenesisSyncCommittee(b.clConfig, clValidators, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis sync committee: %w", err)
	}

	proposers, err := beaconutils.GetGenesisProposers(b.clConfig, clValidators, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate proposer lookahead: %w", err)
	}

	ptcWindow, err := beaconutils.GetGenesisPTCWindow(b.clConfig, clValidators, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate PTC window: %w", err)
	}

	slotsPerEpoch := b.clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)
	emptyBuilderPendingPayments := make([]*gloas.BuilderPendingPayment, slotsPerEpoch*2)
	for i := range slotsPerEpoch * 2 {
		emptyBuilderPendingPayments[i] = &gloas.BuilderPendingPayment{
			Weight: 0,
			Withdrawal: &gloas.BuilderPendingWithdrawal{
				FeeRecipient: bellatrix.ExecutionAddress{},
				Amount:       0,
				BuilderIndex: 0,
			},
		}
	}

	genesisDelay := b.clConfig.GetUintDefault("GENESIS_DELAY", 604800)
	blocksPerHistoricalRoot := b.clConfig.GetUintDefault("SLOTS_PER_HISTORICAL_ROOT", 8192)
	epochsPerSlashingVector := b.clConfig.GetUintDefault("EPOCHS_PER_SLASHINGS_VECTOR", 8192)

	minGenesisTime := b.clConfig.GetUintDefault("MIN_GENESIS_TIME", 0)
	if minGenesisTime == 0 {
		minGenesisTime = genesisBlock.Time()
	}

	genesisState := &gloas.BeaconState{
		GenesisTime:           minGenesisTime + genesisDelay,
		GenesisValidatorsRoot: validatorsRoot,
		Fork:                  GetStateForkConfig(spec.DataVersionFulu, b.clConfig),
		LatestBlockHeader: &phase0.BeaconBlockHeader{
			BodyRoot: genesisBlockBodyRoot,
		},
		BlockRoots: make([]phase0.Root, blocksPerHistoricalRoot),
		StateRoots: make([]phase0.Root, blocksPerHistoricalRoot),
		ETH1Data: &phase0.ETH1Data{
			DepositRoot: depositRoot,
			BlockHash:   genesisBlockHash[:],
		},
		JustificationBits:            make([]byte, 1),
		PreviousJustifiedCheckpoint:  &phase0.Checkpoint{},
		CurrentJustifiedCheckpoint:   &phase0.Checkpoint{},
		FinalizedCheckpoint:          &phase0.Checkpoint{},
		RANDAOMixes:                  beaconutils.SeedRandomMixes(phase0.Hash32(genesisBlockHash), b.clConfig),
		Validators:                   clValidators,
		Balances:                     beaconutils.GetGenesisBalances(b.clConfig, validators),
		Slashings:                    make([]phase0.Gwei, epochsPerSlashingVector),
		PreviousEpochParticipation:   make([]altair.ParticipationFlags, len(clValidators)),
		CurrentEpochParticipation:    make([]altair.ParticipationFlags, len(clValidators)),
		InactivityScores:             make([]uint64, len(clValidators)),
		CurrentSyncCommittee:         syncCommittee,
		NextSyncCommittee:            syncCommittee,
		ProposerLookahead:            proposers,
		Builders:                     clBuilders,
		ExecutionPayloadAvailability: make([]uint8, (blocksPerHistoricalRoot+7)/8),
		BuilderPendingPayments:       emptyBuilderPendingPayments,
		LatestBlockHash:              phase0.Hash32(genesisBlockHash),
		PTCWindow:                    ptcWindow,
	}

	versionedState := &spec.VersionedBeaconState{
		Version: spec.DataVersionGloas,
		Gloas:   genesisState,
	}

	logrus.Infof("genesis version: gloas")
	logrus.Infof("genesis time: %v", genesisState.GenesisTime)
	logrus.Infof("genesis validators root: 0x%x", genesisState.GenesisValidatorsRoot)

	return versionedState, nil
}

func (b *gloasBuilder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != spec.DataVersionGloas {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(state.Gloas)
	case http.ContentTypeJSON:
		return state.Gloas.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
