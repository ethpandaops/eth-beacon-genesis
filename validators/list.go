package validators

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// fileSource is the Source tag assigned to validators loaded from the
// additional-validators file.
const fileSource = "additional-validators"

func LoadValidatorsFromFile(validatorsConfigPath string) ([]*Validator, error) {
	validatorsFile, err := os.Open(validatorsConfigPath)
	if err != nil {
		return nil, err
	}

	defer validatorsFile.Close()

	validators := make([]*Validator, 0)
	pubkeyMap := map[string]int{}

	scanner := bufio.NewScanner(validatorsFile)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		lineParts := strings.Split(line, ":")

		// Public key
		pubKey, err := hex.DecodeString(strings.ReplaceAll(lineParts[0], "0x", ""))
		if err != nil {
			return nil, err
		}

		if len(pubKey) != 48 {
			return nil, fmt.Errorf("invalid pubkey (invalid length) on line %v", lineNum)
		}

		if pubkeyMap[string(pubKey)] != 0 {
			return nil, fmt.Errorf("duplicate pubkey on line %v and %v", pubkeyMap[string(pubKey)], lineNum)
		}

		pubkeyMap[string(pubKey)] = lineNum
		validatorEntry := &Validator{
			PublicKey:             phase0.BLSPubKey(pubKey),
			WithdrawalCredentials: make([]byte, 32),
			Source:                fileSource,
			SourceKeyIndex:        uint64(len(validators)),
		}

		// Withdrawal credentials
		withdrawalCred, err := hex.DecodeString(strings.ReplaceAll(lineParts[1], "0x", ""))
		if err != nil {
			return nil, err
		}

		if len(withdrawalCred) != 32 {
			return nil, fmt.Errorf("invalid withdrawal credentials (invalid length) on line %v", lineNum)
		}

		switch withdrawalCred[0] {
		case 0x00:
		case 0x01, 0x02, 0x03:
			if !bytes.Equal(withdrawalCred[1:12], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
				return nil, fmt.Errorf("invalid withdrawal credentials (invalid 0x01/0x02/0x03 cred) on line %v", lineNum)
			}
		default:
			return nil, fmt.Errorf("invalid withdrawal credentials (invalid type) on line %v", lineNum)
		}

		copy(validatorEntry.WithdrawalCredentials, withdrawalCred)

		// Validator balance
		if len(lineParts) > 2 {
			balance, err := strconv.ParseUint(lineParts[2], 10, 64)
			if err != nil {
				return nil, err
			}

			validatorEntry.Balance = &balance
		}

		validators = append(validators, validatorEntry)
	}

	return validators, nil
}
