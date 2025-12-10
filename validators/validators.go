package validators

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ValidatorStatus uint8

const (
	ValidatorStatusActive ValidatorStatus = iota
	ValidatorStatusSlashed
	ValidatorStatusExited
)

type Validator struct {
	PublicKey             phase0.BLSPubKey
	WithdrawalCredentials []byte
	Balance               *uint64
	Status                ValidatorStatus
}
