package generator

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"

	"github.com/ethpandaops/eth-beacon-genesis/config"
	"github.com/ethpandaops/eth-beacon-genesis/utils"
	"github.com/ethpandaops/eth-beacon-genesis/validators"
	dynssz "github.com/pk910/dynamic-ssz"
)

type electraBuilder struct {
	elGenesis       *core.Genesis
	clConfig        *config.Config
	shadowForkBlock *types.Block
	validators      []*validators.Validator
}

func NewElectraBuilder(elGenesis *core.Genesis, clConfig *config.Config) GenesisBuilder {
	return &electraBuilder{
		elGenesis: elGenesis,
		clConfig:  clConfig,
	}
}

func (b *electraBuilder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *electraBuilder) AddValidators(validators []*validators.Validator) {
	b.validators = append(b.validators, validators...)
}

func (b *electraBuilder) BuildState() (*spec.VersionedBeaconState, error) {
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
		root, err := utils.ComputeWithdrawalsRoot(genesisBlock.Withdrawals(), b.clConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to compute withdrawals root: %w", err)
		}
		withdrawalsRoot = root
	}

	transactionsRoot, err := utils.ComputeTransactionsRoot(genesisBlock.Transactions(), b.clConfig)
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

	depositRoot, err := utils.ComputeDepositRoot(b.clConfig)
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
	}
	genesisBlockBodyRoot, err := genesisBlockBody.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to compute genesis block body root: %w", err)
	}

	clValidators, validatorsRoot := utils.GetGenesisValidators(b.clConfig, b.validators)

	syncCommittee, err := utils.GetGenesisSyncCommittee(b.clConfig, clValidators, phase0.Hash32(genesisBlockHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get genesis sync committee: %w", err)
	}

	genesisDelay := b.clConfig.GetUintDefault("GENESIS_DELAY", 604800)
	blocksPerHistoricalRoot := b.clConfig.GetUintDefault("SLOTS_PER_HISTORICAL_ROOT", 8192)
	epochsPerSlashingVector := b.clConfig.GetUintDefault("EPOCHS_PER_SLASHINGS_VECTOR", 8192)

	genesisState := &electra.BeaconState{
		GenesisTime:           genesisBlock.Time() + genesisDelay,
		GenesisValidatorsRoot: validatorsRoot,
		Fork:                  GetStateForkConfig(spec.DataVersionElectra, b.clConfig),
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
		RANDAOMixes:                  utils.SeedRandomMixes(phase0.Hash32(genesisBlockHash), b.clConfig),
		Validators:                   clValidators,
		Balances:                     utils.GetGenesisBalances(clValidators),
		Slashings:                    make([]phase0.Gwei, epochsPerSlashingVector),
		PreviousEpochParticipation:   make([]altair.ParticipationFlags, len(clValidators)),
		CurrentEpochParticipation:    make([]altair.ParticipationFlags, len(clValidators)),
		InactivityScores:             make([]uint64, len(clValidators)),
		CurrentSyncCommittee:         syncCommittee,
		NextSyncCommittee:            syncCommittee,
		LatestExecutionPayloadHeader: execHeader,
	}

	versionedState := &spec.VersionedBeaconState{
		Version: spec.DataVersionElectra,
		Electra: genesisState,
	}

	return versionedState, nil
}

func (b *electraBuilder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != spec.DataVersionElectra {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	switch contentType {
	case http.ContentTypeSSZ:
		spec := b.clConfig.GetSpecs()
		dynSsz := dynssz.NewDynSsz(spec)
		return dynSsz.MarshalSSZ(state.Electra)
	case http.ContentTypeJSON:
		return state.Electra.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}
