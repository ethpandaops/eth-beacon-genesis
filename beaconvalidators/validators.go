package beaconvalidators

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type Validator struct {
	PublicKey             phase0.BLSPubKey
	WithdrawalCredentials []byte
	Balance               *uint64
}
