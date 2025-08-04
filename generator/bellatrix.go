package generator

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/eth-beacon-genesis/beaconutils"
	"github.com/ethpandaops/eth-beacon-genesis/config"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
	dynssz "github.com/pk910/dynamic-ssz"
)

type bellatrixBuilder struct {
	elGenesis       *core.Genesis
	clConfig        *config.Config
	dynSsz          *dynssz.DynSsz
	shadowForkBlock *types.Block
	validators      []*validators.Validator
}

func NewBellatrixBuilder(elGenesis *core.Genesis, clConfig *config.Config) GenesisBuilder {
	return &bellatrixBuilder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
		dynSsz:    beaconutils.GetDynSSZ(clConfig),
	}
}

func (b *bellatrixBuilder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *bellatrixBuilder) AddValidators(val []*validators.Validator) {
	b.validators = append(b.validators, val...)
}

func (b *bellatrixBuilder) BuildState() (*spec.VersionedBeaconState, error) {
	genesisBlock := b.shadowForkBlock
	if genesisBlock == nil {
		genesisBlock = b.elGenesis.ToBlock()
	}

	genesisBlockHash := genesisBlock.Hash()

	extra := genesisBlock.Extra()
	if len(extra) > 32 {
		return nil, fmt.Errorf("extra data is %d bytes, max is %d", len(extra), 32)
	}

	baseFee, _ := uint256.FromBig(genesisBlock.BaseFee())

	transactionsRoot, err := beaconutils.ComputeTransactionsRoot(genesisBlock.Transactions(), b.clConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to compute transactions root: %w", err)
	}

	baseFeeBytes := baseFee.Bytes32()
	for i, j := 0, len(baseFeeBytes)-1; i < j; i, j = i+1, j-1 {
		baseFeeBytes[i], baseFeeBytes[j] = baseFeeBytes[j], baseFeeBytes[i]
	}

	execHeader := &bellatrix.ExecutionPayloadHeader{
		ParentHash:       phase0.Hash32(genesisBlock.ParentHash()),
		FeeRecipient:     bellatrix.ExecutionAddress(genesisBlock.Coinbase()),
		StateRoot:        phase0.Root(genesisBlock.Root()),
		ReceiptsRoot:     phase0.Root(genesisBlock.ReceiptHash()),
		LogsBloom:        genesisBlock.Bloom(),
		BlockNumber:      genesisBlock.NumberU64(),
		GasLimit:         genesisBlock.GasLimit(),
		GasUsed:          genesisBlock.GasUsed(),
		Timestamp:        genesisBlock.Time(),
		ExtraData:        extra,
		BaseFeePerGas:    baseFeeBytes,
		BlockHash:        phase0.Hash32(genesisBlockHash),
		TransactionsRoot: transactionsRoot,
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

	genesisBlockBody := &bellatrix.BeaconBlockBody{
		ETH1Data: &phase0.ETH1Data{
			BlockHash: make([]byte, 32),
		},
		SyncAggregate: &altair.SyncAggregate{
			SyncCommitteeBits: make([]byte, syncCommitteeMaskBytes),
		},
		ExecutionPayload: &bellatrix.ExecutionPayload{},
	}

	genesisBlockBodyRoot, err := b.dynSsz.HashTreeRoot(genesisBlockBody)
	if err != nil {
		return nil, fmt.Errorf("failed to compute genesis block body root: %w", err)
	}

	clValidators, validatorsRoot := beaconutils.GetGenesisValidators(b.clConfig, b.validators)

	syncCommittee, err := beaconutils.GetGenesisSyncCommittee(b.clConfig, clValidators, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis sync committee: %w", err)
	}

	genesisDelay := b.clConfig.GetUintDefault("GENESIS_DELAY", 604800)
	blocksPerHistoricalRoot := b.clConfig.GetUintDefault("SLOTS_PER_HISTORICAL_ROOT", 8192)
	epochsPerSlashingVector := b.clConfig.GetUintDefault("EPOCHS_PER_SLASHINGS_VECTOR", 8192)

	minGenesisTime := b.clConfig.GetUintDefault("MIN_GENESIS_TIME", 0)
	if minGenesisTime == 0 {
		minGenesisTime = genesisBlock.Time()
	}

	genesisState := &bellatrix.BeaconState{
		GenesisTime:           minGenesisTime + genesisDelay,
		GenesisValidatorsRoot: validatorsRoot,
		Fork:                  GetStateForkConfig(spec.DataVersionBellatrix, b.clConfig),
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
		Balances:                     beaconutils.GetGenesisBalances(b.clConfig, b.validators),
		Slashings:                    make([]phase0.Gwei, epochsPerSlashingVector),
		PreviousEpochParticipation:   make([]altair.ParticipationFlags, len(clValidators)),
		CurrentEpochParticipation:    make([]altair.ParticipationFlags, len(clValidators)),
		InactivityScores:             make([]uint64, len(clValidators)),
		CurrentSyncCommittee:         syncCommittee,
		NextSyncCommittee:            syncCommittee,
		LatestExecutionPayloadHeader: execHeader,
	}

	versionedState := &spec.VersionedBeaconState{
		Version:   spec.DataVersionBellatrix,
		Bellatrix: genesisState,
	}

	logrus.Infof("genesis version: bellatrix")
	logrus.Infof("genesis time: %v", genesisState.GenesisTime)
	logrus.Infof("genesis validators root: 0x%x", genesisState.GenesisValidatorsRoot)

	return versionedState, nil
}

func (b *bellatrixBuilder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != spec.DataVersionBellatrix {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(state.Bellatrix)
	case http.ContentTypeJSON:
		return state.Bellatrix.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
