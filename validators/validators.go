package validators

import (
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

type ValidatorStatus uint8

const (
	ValidatorStatusActive ValidatorStatus = iota
	ValidatorStatusSlashed
	ValidatorStatusExited
)

type PQKeyMaterial struct {
	XMSSPublicKey [32]byte
	VRFRoot       [32]byte
}

type Validator struct {
	PublicKey             phase0.BLSPubKey
	WithdrawalCredentials []byte
	Balance               *uint64
	Status                ValidatorStatus
	PQKeys                *PQKeyMaterial

	// Source identifies where the key originated
	Source         string
	SourceKeyIndex uint64
}

func (v *Validator) UseXMSSPublicKey() {
	v.PublicKey = phase0.BLSPubKey{}
	copy(v.PublicKey[:], v.PQKeys.XMSSPublicKey[:])
}
