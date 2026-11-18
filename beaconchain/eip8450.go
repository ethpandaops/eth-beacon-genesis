package beaconchain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	bitfield "github.com/OffchainLabs/go-bitfield"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/go-eth2-client/http"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/altair"
	"github.com/ethpandaops/go-eth2-client/spec/capella"
	"github.com/ethpandaops/go-eth2-client/spec/electra"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/heze"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/eth-beacon-genesis/beaconconfig"
	"github.com/ethpandaops/eth-beacon-genesis/beaconutils"
)

const farFutureEpoch = phase0.Epoch(^uint64(0))

type PQKeyMaterial struct {
	XMSSPublicKey [32]byte    `ssz-size:"32"`
	VRFRoot       phase0.Root `ssz-size:"32"`
}

type PQKeyRegistration struct {
	Index               phase0.ValidatorIndex
	KeyMaterial         *PQKeyMaterial
	PreviousChangeEpoch phase0.Epoch
}

type SignedPQKeyRegistration struct {
	Message    *PQKeyRegistration
	Possession [2404]byte          `ssz-size:"2404"`
	Signature  phase0.BLSSignature `ssz-size:"96"`
}

type EIP8450BeaconBlockBody struct {
	RANDAOReveal              phase0.BLSSignature                   `ssz-index:"0"`
	Graffiti                  [32]byte                              `ssz-index:"2"`
	ProposerSlashings         []*phase0.ProposerSlashing            `ssz-index:"3"  ssz-type:"progressive-list"`
	AttesterSlashings         []*gloas.AttesterSlashing             `ssz-index:"4"  ssz-type:"progressive-list"`
	Attestations              []*gloas.Attestation                  `ssz-index:"5"  ssz-type:"progressive-list"`
	VoluntaryExits            []*phase0.SignedVoluntaryExit         `ssz-index:"7"  ssz-type:"progressive-list"`
	SyncAggregate             *altair.SyncAggregate                 `ssz-index:"8"`
	BLSToExecutionChanges     []*capella.SignedBLSToExecutionChange `ssz-index:"9"  ssz-type:"progressive-list"`
	SignedExecutionPayloadBid *heze.SignedExecutionPayloadBid       `ssz-index:"10"`
	PayloadAttestations       []*gloas.PayloadAttestation           `ssz-index:"11" ssz-type:"progressive-list"`
	ParentExecutionRequests   *gloas.ExecutionRequests              `ssz-index:"12"`
	PQKeyRegistrations        []*SignedPQKeyRegistration            `ssz-index:"13" ssz-type:"progressive-list"`
}

type EIP8450BeaconState struct {
	GenesisTime                       uint64                              `ssz-index:"0"`
	GenesisValidatorsRoot             phase0.Root                         `ssz-index:"1" ssz-size:"32"`
	Slot                              phase0.Slot                         `ssz-index:"2"`
	Fork                              *phase0.Fork                        `ssz-index:"3"`
	LatestBlockHeader                 *phase0.BeaconBlockHeader           `ssz-index:"4"`
	BlockRoots                        []phase0.Root                       `ssz-index:"5" dynssz-size:"SLOTS_PER_HISTORICAL_ROOT,32" ssz-size:"8192,32"`
	StateRoots                        []phase0.Root                       `ssz-index:"6" dynssz-size:"SLOTS_PER_HISTORICAL_ROOT,32" ssz-size:"8192,32"`
	HistoricalRoots                   []phase0.Root                       `ssz-index:"7" ssz-max:"16777216"                         ssz-size:"?,32"`
	Validators                        []*phase0.Validator                 `ssz-index:"11" ssz-type:"progressive-list"`
	Balances                          []phase0.Gwei                       `ssz-index:"12" ssz-type:"progressive-list"`
	RANDAOMixes                       []phase0.Root                       `ssz-index:"13" dynssz-size:"EPOCHS_PER_HISTORICAL_VECTOR,32" ssz-size:"65536,32"`
	Slashings                         []phase0.Gwei                       `ssz-index:"14" dynssz-size:"EPOCHS_PER_SLASHINGS_VECTOR"     ssz-size:"8192"`
	PreviousEpochParticipation        []altair.ParticipationFlags         `ssz-index:"15" ssz-type:"progressive-list"`
	CurrentEpochParticipation         []altair.ParticipationFlags         `ssz-index:"16" ssz-type:"progressive-list"`
	JustificationBits                 bitfield.Bitvector4                 `ssz-index:"17" ssz-size:"1"`
	PreviousJustifiedCheckpoint       *phase0.Checkpoint                  `ssz-index:"18"`
	CurrentJustifiedCheckpoint        *phase0.Checkpoint                  `ssz-index:"19"`
	FinalizedCheckpoint               *phase0.Checkpoint                  `ssz-index:"20"`
	InactivityScores                  []uint64                            `ssz-index:"21" ssz-type:"progressive-list"`
	CurrentSyncCommittee              *altair.SyncCommittee               `ssz-index:"22"`
	NextSyncCommittee                 *altair.SyncCommittee               `ssz-index:"23"`
	LatestBlockHash                   phase0.Hash32                       `ssz-index:"24" ssz-size:"32"`
	NextWithdrawalIndex               capella.WithdrawalIndex             `ssz-index:"25"`
	NextWithdrawalValidatorIndex      phase0.ValidatorIndex               `ssz-index:"26"`
	HistoricalSummaries               []*capella.HistoricalSummary        `ssz-index:"27" ssz-max:"16777216"`
	DepositBalanceToConsume           phase0.Gwei                         `ssz-index:"29"`
	ExitBalanceToConsume              phase0.Gwei                         `ssz-index:"30"`
	EarliestExitEpoch                 phase0.Epoch                        `ssz-index:"31"`
	ConsolidationBalanceToConsume     phase0.Gwei                         `ssz-index:"32"`
	EarliestConsolidationEpoch        phase0.Epoch                        `ssz-index:"33"`
	PendingDeposits                   []*electra.PendingDeposit           `ssz-index:"34" ssz-type:"progressive-list"`
	PendingPartialWithdrawals         []*electra.PendingPartialWithdrawal `ssz-index:"35" ssz-type:"progressive-list"`
	PendingConsolidations             []*electra.PendingConsolidation     `ssz-index:"36" ssz-type:"progressive-list"`
	ProposerLookahead                 []phase0.ValidatorIndex             `ssz-index:"37" dynssz-size:"(MIN_SEED_LOOKAHEAD+1)*SLOTS_PER_EPOCH" ssz-size:"64"`
	Builders                          []*gloas.Builder                    `ssz-index:"38" ssz-type:"progressive-list"`
	NextWithdrawalBuilderIndex        gloas.BuilderIndex                  `ssz-index:"39"`
	ExecutionPayloadAvailability      []uint8                             `ssz-index:"40" dynssz-size:"SLOTS_PER_HISTORICAL_ROOT/8"      ssz-size:"1024"`
	BuilderPendingPayments            []*gloas.BuilderPendingPayment      `ssz-index:"41" dynssz-size:"SLOTS_PER_EPOCH*2"                ssz-size:"64"`
	BuilderPendingWithdrawals         []*gloas.BuilderPendingWithdrawal   `ssz-index:"42" ssz-type:"progressive-list"`
	LatestExecutionPayloadBid         *heze.ExecutionPayloadBid           `ssz-index:"43"`
	PayloadExpectedWithdrawals        []*capella.Withdrawal               `ssz-index:"44" ssz-type:"progressive-list"`
	PTCWindow                         [][]phase0.ValidatorIndex           `ssz-index:"45" dynssz-size:"(2+MIN_SEED_LOOKAHEAD)*SLOTS_PER_EPOCH,PTC_SIZE" ssz-size:"96,512"`
	PostQuantumKeyMaterial            []*PQKeyMaterial                    `ssz-index:"46" ssz-type:"progressive-list"`
	PostQuantumKeyChangeEpochs        []phase0.Epoch                      `ssz-index:"47" ssz-type:"progressive-list"`
	BuilderPostQuantumKeyMaterial     []*PQKeyMaterial                    `ssz-index:"48" ssz-type:"progressive-list"`
	BuilderPostQuantumKeyChangeEpochs []phase0.Epoch                      `ssz-index:"49" ssz-type:"progressive-list"`
}

type eip8450Builder struct {
	hezeBuilder
}

func NewEIP8450Builder(elGenesis *core.Genesis, clConfig *beaconconfig.Config) BeaconGenesisBuilder {
	return &eip8450Builder{
		hezeBuilder: hezeBuilder{
			elGenesis: elGenesis,
			clConfig:  clConfig,
			dynSsz:    beaconutils.GetDynSSZ(clConfig),
		},
	}
}

func (b *eip8450Builder) SetShadowForkBlock(block *types.Block) {
	b.shadowForkBlock = block
}

func (b *eip8450Builder) BuildState() (*spec.VersionedBeaconState, error) {
	genesisState, err := b.buildState(DataVersionEIP8450, func(body *heze.BeaconBlockBody) any {
		pqBody := &EIP8450BeaconBlockBody{}
		copyFields(pqBody, body)

		return pqBody
	})
	if err != nil {
		return nil, err
	}

	logrus.Infof("genesis version: eip8450")
	logrus.Infof("genesis time: %v", genesisState.GenesisTime)
	logrus.Infof("genesis validators root: 0x%x", genesisState.GenesisValidatorsRoot)

	return &spec.VersionedBeaconState{
		Version: spec.DataVersionHeze,
		Heze:    genesisState,
	}, nil
}

func (b *eip8450Builder) pqState(hezeState *heze.BeaconState) *EIP8450BeaconState {
	genesisBuilders, genesisVals := beaconutils.SeparateBuildersFromValidators(b.validators)

	state := &EIP8450BeaconState{
		PostQuantumKeyMaterial:            make([]*PQKeyMaterial, len(genesisVals)),
		PostQuantumKeyChangeEpochs:        make([]phase0.Epoch, len(genesisVals)),
		BuilderPostQuantumKeyMaterial:     make([]*PQKeyMaterial, len(genesisBuilders)),
		BuilderPostQuantumKeyChangeEpochs: make([]phase0.Epoch, len(genesisBuilders)),
	}
	copyFields(state, hezeState)

	for i, val := range genesisVals {
		state.PostQuantumKeyMaterial[i] = &PQKeyMaterial{}
		state.PostQuantumKeyChangeEpochs[i] = farFutureEpoch

		if val.PQKeys != nil {
			state.PostQuantumKeyMaterial[i].XMSSPublicKey = val.PQKeys.XMSSPublicKey
			state.PostQuantumKeyMaterial[i].VRFRoot = val.PQKeys.VRFRoot
			state.PostQuantumKeyChangeEpochs[i] = 0
		}
	}

	for i, builder := range genesisBuilders {
		state.BuilderPostQuantumKeyMaterial[i] = &PQKeyMaterial{}
		state.BuilderPostQuantumKeyChangeEpochs[i] = hezeState.Builders[i].DepositEpoch

		if builder.PQKeys != nil {
			state.BuilderPostQuantumKeyMaterial[i].XMSSPublicKey = builder.PQKeys.XMSSPublicKey
		}
	}

	return state
}

func (b *eip8450Builder) Serialize(state *spec.VersionedBeaconState, contentType http.ContentType) ([]byte, error) {
	if state.Version != spec.DataVersionHeze {
		return nil, fmt.Errorf("unsupported version: %s", state.Version)
	}

	pqState := b.pqState(state.Heze)

	switch contentType {
	case http.ContentTypeSSZ:
		return b.dynSsz.MarshalSSZ(pqState)
	case http.ContentTypeJSON:
		hezeJSON, err := state.Heze.MarshalJSON()
		if err != nil {
			return nil, err
		}

		pqJSON, err := json.Marshal(struct {
			PostQuantumKeyMaterial            []pqKeyMaterialJSON `json:"post_quantum_key_material"`
			PostQuantumKeyChangeEpochs        []string            `json:"post_quantum_key_change_epochs"`
			BuilderPostQuantumKeyMaterial     []pqKeyMaterialJSON `json:"builder_post_quantum_key_material"`
			BuilderPostQuantumKeyChangeEpochs []string            `json:"builder_post_quantum_key_change_epochs"`
		}{
			PostQuantumKeyMaterial:            pqKeyMaterialToJSON(pqState.PostQuantumKeyMaterial),
			PostQuantumKeyChangeEpochs:        epochsToJSON(pqState.PostQuantumKeyChangeEpochs),
			BuilderPostQuantumKeyMaterial:     pqKeyMaterialToJSON(pqState.BuilderPostQuantumKeyMaterial),
			BuilderPostQuantumKeyChangeEpochs: epochsToJSON(pqState.BuilderPostQuantumKeyChangeEpochs),
		})
		if err != nil {
			return nil, err
		}

		return append(append(hezeJSON[:len(hezeJSON)-1:len(hezeJSON)-1], ','), pqJSON[1:]...), nil
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

type pqKeyMaterialJSON struct {
	XMSSPublicKey string `json:"xmss_public_key"`
	VRFRoot       string `json:"vrf_root"`
}

func pqKeyMaterialToJSON(entries []*PQKeyMaterial) []pqKeyMaterialJSON {
	res := make([]pqKeyMaterialJSON, len(entries))
	for i, entry := range entries {
		res[i] = pqKeyMaterialJSON{
			XMSSPublicKey: fmt.Sprintf("%#x", entry.XMSSPublicKey),
			VRFRoot:       fmt.Sprintf("%#x", entry.VRFRoot),
		}
	}

	return res
}

func epochsToJSON(epochs []phase0.Epoch) []string {
	res := make([]string, len(epochs))
	for i, epoch := range epochs {
		res[i] = strconv.FormatUint(uint64(epoch), 10)
	}

	return res
}

func copyFields(dst, src any) {
	srcVal := reflect.ValueOf(src).Elem()
	dstVal := reflect.ValueOf(dst).Elem()

	for i := range srcVal.NumField() {
		dstVal.FieldByName(srcVal.Type().Field(i).Name).Set(srcVal.Field(i))
	}
}
