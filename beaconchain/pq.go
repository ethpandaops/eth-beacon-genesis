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
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/beaconutils"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
	dynssz "github.com/pk910/dynamic-ssz"
)

type pqBuilder struct {
	elGenesis       *core.Genesis
	clConfig        *beaconconfig.Config
	dynSsz          *dynssz.DynSsz
	shadowForkBlock *types.Block
	validators      []*validators.Validator
	state           *PQBeaconState
}

func NewPQBuilder(elGenesis *core.Genesis, clConfig *beaconconfig.Config) BeaconGenesisBuilder {
	return &pqBuilder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
		dynSsz:    beaconutils.GetDynSSZ(clConfig),
	}
}

func (b *pqBuilder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *pqBuilder) AddValidators(val []*validators.Validator) {
	b.validators = append(b.validators, val...)
}

func (b *pqBuilder) BuildState() (*spec.VersionedBeaconState, error) {
	genesisBlock := b.shadowForkBlock
	if genesisBlock == nil {
		genesisBlock = b.elGenesis.ToBlock()
	}

	genesisBlockHash := genesisBlock.Hash()

	extra := genesisBlock.Extra()
	if len(extra) > 32 {
		return nil, fmt.Errorf("extra data is %d bytes, max is %d", len(extra), 32)
	}

	for i, val := range b.validators {
		if val.PQKeys == nil {
			return nil, fmt.Errorf("validator %d has no leanXMSS key", i)
		}
	}

	depositRoot, err := beaconutils.ComputeDepositRoot(b.clConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to compute deposit root: %w", err)
	}

	emptyExecutionRequests := &PQExecutionRequests{}

	executionRequestsRoot, err := b.dynSsz.HashTreeRoot(emptyExecutionRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to compute empty execution requests root: %w", err)
	}

	genesisBlockBody := &PQBeaconBlockBody{
		ETH1Data: &phase0.ETH1Data{
			BlockHash: make([]byte, 32),
		},
		SignedExecutionPayloadBid: &PQSignedExecutionPayloadBid{
			Message: &gloas.ExecutionPayloadBid{
				ParentBlockHash:       phase0.Hash32(genesisBlockHash),
				ExecutionRequestsRoot: executionRequestsRoot,
			},
		},
		ParentExecutionRequests: emptyExecutionRequests,
	}

	genesisBlockBodyRoot, err := b.dynSsz.HashTreeRoot(genesisBlockBody)
	if err != nil {
		return nil, fmt.Errorf("failed to compute genesis block body root: %w", err)
	}

	genesisBuilders, genesisVals := beaconutils.SeparateBuildersFromValidators(b.validators)
	registry, _ := beaconutils.GetGenesisValidators(b.clConfig, genesisVals)

	clValidators := make([]*PQValidator, len(registry))
	for i, validator := range registry {
		clValidators[i] = &PQValidator{
			Pubkey:                     genesisVals[i].PQKeys.XMSSPublicKey,
			VRFRoot:                    genesisVals[i].PQKeys.VRFRoot,
			WithdrawalCredentials:      validator.WithdrawalCredentials,
			EffectiveBalance:           validator.EffectiveBalance,
			Slashed:                    validator.Slashed,
			ActivationEligibilityEpoch: validator.ActivationEligibilityEpoch,
			ActivationEpoch:            validator.ActivationEpoch,
			ExitEpoch:                  validator.ExitEpoch,
			WithdrawableEpoch:          validator.WithdrawableEpoch,
		}
	}

	validatorsRoot, err := b.dynSsz.HashTreeRoot(&struct {
		Validators []*PQValidator `ssz-type:"progressive-list"`
	}{clValidators})
	if err != nil {
		return nil, fmt.Errorf("failed to compute validators root: %w", err)
	}

	builders := beaconutils.GetGenesisBuilders(b.clConfig, genesisBuilders)

	clBuilders := make([]*PQBuilder, len(builders))
	for i, builder := range builders {
		clBuilders[i] = &PQBuilder{
			Pubkey:            genesisBuilders[i].PQKeys.XMSSPublicKey,
			Version:           builder.Version,
			ExecutionAddress:  builder.ExecutionAddress,
			Balance:           builder.Balance,
			DepositEpoch:      builder.DepositEpoch,
			WithdrawableEpoch: builder.WithdrawableEpoch,
		}
	}

	proposers, err := beaconutils.GetGenesisProposers(b.clConfig, registry, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate proposer lookahead: %w", err)
	}

	ptcWindow, err := beaconutils.GetGenesisPTCWindow(b.clConfig, registry, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate PTC window: %w", err)
	}

	aggregationCommitteeWindow, err := beaconutils.GetGenesisAggregationCommitteeWindow(b.clConfig, registry, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate aggregation committee window: %w", err)
	}

	slotsPerEpoch := b.clConfig.GetUintDefault("SLOTS_PER_EPOCH", 32)

	emptyBuilderPendingPayments := make([]*gloas.BuilderPendingPayment, slotsPerEpoch*2)
	for i := range slotsPerEpoch * 2 {
		emptyBuilderPendingPayments[i] = &gloas.BuilderPendingPayment{
			Withdrawal: &gloas.BuilderPendingWithdrawal{
				FeeRecipient: bellatrix.ExecutionAddress{},
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

	forkVersion, _ := b.clConfig.GetBytes("PQ_FORK_VERSION")
	previousForkVersion, _ := b.clConfig.GetBytes("GLOAS_FORK_VERSION")

	b.state = &PQBeaconState{
		GenesisTime:           minGenesisTime + genesisDelay,
		GenesisValidatorsRoot: validatorsRoot,
		Fork: &phase0.Fork{
			CurrentVersion:  phase0.Version(forkVersion),
			PreviousVersion: phase0.Version(previousForkVersion),
		},
		LatestBlockHeader: &phase0.BeaconBlockHeader{
			BodyRoot: genesisBlockBodyRoot,
		},
		BlockRoots:      make([]phase0.Root, blocksPerHistoricalRoot),
		StateRoots:      make([]phase0.Root, blocksPerHistoricalRoot),
		HistoricalRoots: []phase0.Root{},
		ETH1Data: &phase0.ETH1Data{
			DepositRoot: depositRoot,
			BlockHash:   genesisBlockHash[:],
		},
		ETH1DataVotes:               []*phase0.ETH1Data{},
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
		HistoricalSummaries:         []*capella.HistoricalSummary{},
		PendingDeposits:             []*electra.PendingDeposit{},
		PendingPartialWithdrawals:   []*electra.PendingPartialWithdrawal{},
		PendingConsolidations:       []*electra.PendingConsolidation{},
		ProposerLookahead:           proposers,
		Builders:                    clBuilders,
		LatestExecutionPayloadBid: &gloas.ExecutionPayloadBid{
			ParentBlockHash:       phase0.Hash32(genesisBlockHash),
			ExecutionRequestsRoot: executionRequestsRoot,
		},
		ExecutionPayloadAvailability: beaconutils.MakeAllOnesBitvector(blocksPerHistoricalRoot),
		BuilderPendingPayments:       emptyBuilderPendingPayments,
		BuilderPendingWithdrawals:    []*gloas.BuilderPendingWithdrawal{},
		PayloadExpectedWithdrawals:   []*capella.Withdrawal{},
		LatestBlockHash:              phase0.Hash32(genesisBlockHash),
		PTCWindow:                    ptcWindow,
		ValidatorPreregistrations:    []*PQValidatorPreregistration{},
		BuilderPreregistrations:      []*PQBuilderPreregistration{},
		AggregationCommitteeWindow:   aggregationCommitteeWindow,
	}

	stateRoot, err := b.dynSsz.HashTreeRoot(b.state)
	if err != nil {
		return nil, fmt.Errorf("failed to compute genesis state root: %w", err)
	}

	logrus.Infof("genesis version: pq")
	logrus.Infof("genesis time: %v", b.state.GenesisTime)
	logrus.Infof("genesis validators root: 0x%x", b.state.GenesisValidatorsRoot)
	logrus.Infof("genesis state root: 0x%x", stateRoot)

	return &spec.VersionedBeaconState{Version: DataVersionPQ}, nil
}

func (b *pqBuilder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != DataVersionPQ || b.state == nil {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(b.state)
	case http.ContentTypeJSON:
		return marshalPQJSON(b.state)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
