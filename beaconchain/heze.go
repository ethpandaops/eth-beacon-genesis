package beaconchain

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/go-eth2-client/http"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/beaconutils"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
	dynssz "github.com/pk910/dynamic-ssz"
)

type hezeBuilder struct {
	elGenesis       *core.Genesis
	clConfig        *beaconconfig.Config
	dynSsz          *dynssz.DynSsz
	shadowForkBlock *types.Block
	validators      []*validators.Validator
}

func NewHezeBuilder(elGenesis *core.Genesis, clConfig *beaconconfig.Config) BeaconGenesisBuilder {
	return &hezeBuilder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
		dynSsz:    beaconutils.GetDynSSZ(clConfig),
	}
}

func (b *hezeBuilder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *hezeBuilder) AddValidators(val []*validators.Validator) {
	b.validators = append(b.validators, val...)
}

func (b *hezeBuilder) BuildState() (*spec.VersionedBeaconState, error) {
	genesisBlock := b.shadowForkBlock
	if genesisBlock == nil {
		genesisBlock = b.elGenesis.ToBlock()
	}

	genesisBlockHash := genesisBlock.Hash()

	extra := genesisBlock.Extra()
	if len(extra) > 32 {
		return nil, fmt.Errorf("extra data is %d bytes, max is %d", len(extra), 32)
	}

	syncCommitteeSize := b.clConfig.GetUintDefault("SYNC_COMMITTEE_SIZE", 512)
	syncCommitteeMaskBytes := syncCommitteeSize / 8

	if syncCommitteeSize%8 != 0 {
		syncCommitteeMaskBytes++
	}

	emptyExecutionRequests := &gloas.ExecutionRequests{}

	executionRequestsRoot, err := b.dynSsz.HashTreeRoot(emptyExecutionRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to compute empty execution requests root: %w", err)
	}

	inclusionListCommitteeSize := b.clConfig.GetUintDefault("INCLUSION_LIST_COMMITTEE_SIZE", 16)
	inclusionListBitsBytes := inclusionListCommitteeSize / 8

	if inclusionListCommitteeSize%8 != 0 {
		inclusionListBitsBytes++
	}

	genesisBlockBody := &heze.BeaconBlockBody{
		SyncAggregate: &altair.SyncAggregate{
			SyncCommitteeBits: make([]byte, syncCommitteeMaskBytes),
		},
		SignedExecutionPayloadBid: &heze.SignedExecutionPayloadBid{
			Message: &heze.ExecutionPayloadBid{
				ParentBlockHash:       phase0.Hash32(genesisBlockHash),
				ExecutionRequestsRoot: executionRequestsRoot,
				InclusionListBits:     make([]byte, inclusionListBitsBytes),
			},
			Signature: phase0.BLSSignature(make([]byte, 96)),
		},
		ParentExecutionRequests: emptyExecutionRequests,
	}

	genesisBlockBodyRoot, err := b.dynSsz.HashTreeRoot(genesisBlockBody)
	if err != nil {
		return nil, fmt.Errorf("failed to compute genesis block body root: %w", err)
	}

	genesisBuilders, genesisVals := beaconutils.SeparateBuildersFromValidators(b.validators)
	clValidators, validatorsRoot := beaconutils.GetGenesisValidators(b.clConfig, genesisVals)
	clBuilders := beaconutils.GetGenesisBuilders(b.clConfig, genesisBuilders)

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

	genesisState := &heze.BeaconState{
		GenesisTime:           minGenesisTime + genesisDelay,
		GenesisValidatorsRoot: validatorsRoot,
		Fork:                  GetStateForkConfig(spec.DataVersionHeze, b.clConfig),
		LatestBlockHeader: &phase0.BeaconBlockHeader{
			BodyRoot: genesisBlockBodyRoot,
		},
		BlockRoots:                  make([]phase0.Root, blocksPerHistoricalRoot),
		StateRoots:                  make([]phase0.Root, blocksPerHistoricalRoot),
		JustificationBits:           make([]byte, 1),
		PreviousJustifiedCheckpoint: &phase0.Checkpoint{},
		CurrentJustifiedCheckpoint:  &phase0.Checkpoint{},
		FinalizedCheckpoint:         &phase0.Checkpoint{},
		RANDAOMixes:                 beaconutils.SeedRandomMixes(phase0.Hash32(genesisBlockHash), b.clConfig),
		Validators:                  clValidators,
		Balances:                    beaconutils.GetGenesisBalances(b.clConfig, genesisVals),
		Slashings:                   make([]phase0.Gwei, epochsPerSlashingVector),
		PreviousEpochParticipation:  make([]altair.ParticipationFlags, len(clValidators)),
		CurrentEpochParticipation:   make([]altair.ParticipationFlags, len(clValidators)),
		InactivityScores:            make([]uint64, len(clValidators)),
		CurrentSyncCommittee:        syncCommittee,
		NextSyncCommittee:           syncCommittee,
		ProposerLookahead:           proposers,
		Builders:                    clBuilders,
		LatestExecutionPayloadBid: &heze.ExecutionPayloadBid{
			ParentBlockHash:       phase0.Hash32(genesisBlockHash),
			ExecutionRequestsRoot: executionRequestsRoot,
			InclusionListBits:     make([]byte, inclusionListBitsBytes),
		},
		ExecutionPayloadAvailability: beaconutils.MakeAllOnesBitvector(blocksPerHistoricalRoot),
		BuilderPendingPayments:       emptyBuilderPendingPayments,
		PayloadExpectedWithdrawals:   []*capella.Withdrawal{},
		LatestBlockHash:              phase0.Hash32(genesisBlockHash),
		PTCWindow:                    ptcWindow,
	}

	versionedState := &spec.VersionedBeaconState{
		Version: spec.DataVersionHeze,
		Heze:    genesisState,
	}

	logrus.Infof("genesis version: heze")
	logrus.Infof("genesis time: %v", genesisState.GenesisTime)
	logrus.Infof("genesis validators root: 0x%x", genesisState.GenesisValidatorsRoot)

	return versionedState, nil
}

func (b *hezeBuilder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != spec.DataVersionHeze {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(state.Heze)
	case http.ContentTypeJSON:
		return state.Heze.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
