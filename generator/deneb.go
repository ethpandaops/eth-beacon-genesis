package generator

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
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

type denebBuilder struct {
	elGenesis       *core.Genesis
	clConfig        *config.Config
	dynSsz          *dynssz.DynSsz
	shadowForkBlock *types.Block
	validators      []*validators.Validator
}

func NewDenebBuilder(elGenesis *core.Genesis, clConfig *config.Config) GenesisBuilder {
	return &denebBuilder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
		dynSsz:    beaconutils.GetDynSSZ(clConfig),
	}
}

func (b *denebBuilder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *denebBuilder) AddValidators(val []*validators.Validator) {
	b.validators = append(b.validators, val...)
}

func (b *denebBuilder) BuildState() (*spec.VersionedBeaconState, error) {
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

	var withdrawalsRoot phase0.Root

	if genesisBlock.Withdrawals() != nil {
		root, err := beaconutils.ComputeWithdrawalsRoot(genesisBlock.Withdrawals(), b.clConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to compute withdrawals root: %w", err)
		}

		withdrawalsRoot = root
	}

	transactionsRoot, err := beaconutils.ComputeTransactionsRoot(genesisBlock.Transactions(), b.clConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to compute transactions root: %w", err)
	}

	if genesisBlock.BlobGasUsed() == nil {
		return nil, fmt.Errorf("execution-layer Block has missing blob-gas-used field")
	}

	if genesisBlock.ExcessBlobGas() == nil {
		return nil, fmt.Errorf("execution-layer Block has missing excess-blob-gas field")
	}

	execHeader := &deneb.ExecutionPayloadHeader{
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
		BaseFeePerGas:    baseFee,
		BlockHash:        phase0.Hash32(genesisBlockHash),
		TransactionsRoot: transactionsRoot,
		WithdrawalsRoot:  withdrawalsRoot,
		BlobGasUsed:      *genesisBlock.BlobGasUsed(),
		ExcessBlobGas:    *genesisBlock.ExcessBlobGas(),
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

	genesisBlockBody := &deneb.BeaconBlockBody{
		ETH1Data: &phase0.ETH1Data{
			BlockHash: make([]byte, 32),
		},
		SyncAggregate: &altair.SyncAggregate{
			SyncCommitteeBits: make([]byte, syncCommitteeMaskBytes),
		},
		ExecutionPayload: &deneb.ExecutionPayload{
			BaseFeePerGas: uint256.NewInt(0),
		},
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

	genesisState := &deneb.BeaconState{
		GenesisTime:           minGenesisTime + genesisDelay,
		GenesisValidatorsRoot: validatorsRoot,
		Fork:                  GetStateForkConfig(spec.DataVersionDeneb, b.clConfig),
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
		Version: spec.DataVersionDeneb,
		Deneb:   genesisState,
	}

	logrus.Infof("genesis version: deneb")
	logrus.Infof("genesis time: %v", genesisState.GenesisTime)
	logrus.Infof("genesis validators root: 0x%x", genesisState.GenesisValidatorsRoot)

	return versionedState, nil
}

func (b *denebBuilder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != spec.DataVersionDeneb {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(state.Deneb)
	case http.ContentTypeJSON:
		return state.Deneb.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
