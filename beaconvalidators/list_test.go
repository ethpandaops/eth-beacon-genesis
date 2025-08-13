package beaconvalidators

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestValidatorsFile(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()

	validatorsFile, err := os.Create(filepath.Join(dir, "validators.txt"))
	if err != nil {
		t.Fatalf("failed to create validators file: %v", err)
	}

	_, err = validatorsFile.WriteString(data)
	if err != nil {
		t.Fatalf("failed to write validators data: %v", err)
	}

	return validatorsFile.Name()
}

func TestLoadValidatorsFromFile_Valid(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf:32000000000
0xace5689384f87725790499fb5261b586d7dfb7d86058f0a909856272ba02df9929dcdb4b1ea529b02b948b3a1dca4d57:0x008aa7b9c37bf27e7c49a3185a3e721c7a02c94da7a0b6ad5f88f1b0477d3b88:64000000000

# balance is optional and defaults to 32000000000:
0xa33dfc09b4031e8c520469024c0ef419cc148f71d7b9501f58f2e54fc644462f208119791e57c5c9b33bf5e47f705060:00b84654c946dc68b353384426a29a3c5d736d9f751c192d5038206e93f79d73

# individual 0x01 or 0x02 credentials can be set:
82fc9f31d6e768c57d09483e788b24444235f64d2cae5f2f8a9dd28b6e8ed6636a5f378febc762cfcd9f8ab808286608:010000000000000000000000CcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC
0xb744b5466a214762ee17621dc4c75d1bba16417e20755f7c9c2485ea518580be50d2c87d70cc4ac393158eb34311c9a2:020000000000000000000000000000000000000000000000000000000000dEaD:64000000000
`)

	validators, err := LoadValidatorsFromFile(validatorsFile)
	if err != nil {
		t.Fatalf("failed to load validators: %v", err)
	}

	if len(validators) != 5 {
		t.Fatalf("expected 5 validators, got %d", len(validators))
	}

	// Validator 0
	if value, _ := hex.DecodeString("9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4"); !bytes.Equal(validators[0].PublicKey[:], value) {
		t.Fatalf("expected validator 0 to have pubkey 0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4, got %s", validators[0].PublicKey.String())
	}

	if value, _ := hex.DecodeString("001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf"); !bytes.Equal(validators[0].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 0 to have withdrawal credentials 0x001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf, got 0x%x", validators[0].WithdrawalCredentials)
	}

	if validators[0].Balance == nil || *validators[0].Balance != 32000000000 {
		t.Fatalf("expected validator 0 to have balance 32000000000, got %d", validators[0].Balance)
	}

	// Validator 1
	if value, _ := hex.DecodeString("ace5689384f87725790499fb5261b586d7dfb7d86058f0a909856272ba02df9929dcdb4b1ea529b02b948b3a1dca4d57"); !bytes.Equal(validators[1].PublicKey[:], value) {
		t.Fatalf("expected validator 1 to have pubkey 0xace5689384f87725790499fb5261b586d7dfb7d86058f0a909856272ba02df9929dcdb4b1ea529b02b948b3a1dca4d57, got %s", validators[1].PublicKey.String())
	}

	if value, _ := hex.DecodeString("008aa7b9c37bf27e7c49a3185a3e721c7a02c94da7a0b6ad5f88f1b0477d3b88"); !bytes.Equal(validators[1].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 1 to have withdrawal credentials 0x008aa7b9c37bf27e7c49a3185a3e721c7a02c94da7a0b6ad5f88f1b0477d3b88, got %s", validators[1].WithdrawalCredentials)
	}

	if validators[1].Balance == nil || *validators[1].Balance != 64000000000 {
		t.Fatalf("expected validator 1 to have balance 64000000000, got %d", validators[1].Balance)
	}

	// Validator 2
	if value, _ := hex.DecodeString("a33dfc09b4031e8c520469024c0ef419cc148f71d7b9501f58f2e54fc644462f208119791e57c5c9b33bf5e47f705060"); !bytes.Equal(validators[2].PublicKey[:], value) {
		t.Fatalf("expected validator 2 to have pubkey 0xa33dfc09b4031e8c520469024c0ef419cc148f71d7b9501f58f2e54fc644462f208119791e57c5c9b33bf5e47f705060, got %s", validators[2].PublicKey.String())
	}

	if value, _ := hex.DecodeString("00b84654c946dc68b353384426a29a3c5d736d9f751c192d5038206e93f79d73"); !bytes.Equal(validators[2].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 2 to have withdrawal credentials 0x00b84654c946dc68b353384426a29a3c5d736d9f751c192d5038206e93f79d73, got %s", validators[2].WithdrawalCredentials)
	}

	if validators[2].Balance != nil {
		t.Fatalf("expected validator 2 to have no balance, got %d", validators[2].Balance)
	}

	// Validator 3
	if value, _ := hex.DecodeString("82fc9f31d6e768c57d09483e788b24444235f64d2cae5f2f8a9dd28b6e8ed6636a5f378febc762cfcd9f8ab808286608"); !bytes.Equal(validators[3].PublicKey[:], value) {
		t.Fatalf("expected validator 3 to have pubkey 0x82fc9f31d6e768c57d09483e788b24444235f64d2cae5f2f8a9dd28b6e8ed6636a5f378febc762cfcd9f8ab808286608, got %s", validators[3].PublicKey.String())
	}

	if value, _ := hex.DecodeString("010000000000000000000000CcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"); !bytes.Equal(validators[3].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 3 to have withdrawal credentials 0x010000000000000000000000CcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC, got %s", validators[3].WithdrawalCredentials)
	}

	if validators[3].Balance != nil {
		t.Fatalf("expected validator 3 to have no balance, got %d", validators[3].Balance)
	}

	// Validator 4
	if value, _ := hex.DecodeString("b744b5466a214762ee17621dc4c75d1bba16417e20755f7c9c2485ea518580be50d2c87d70cc4ac393158eb34311c9a2"); !bytes.Equal(validators[4].PublicKey[:], value) {
		t.Fatalf("expected validator 4 to have pubkey 0xb744b5466a214762ee17621dc4c75d1bba16417e20755f7c9c2485ea518580be50d2c87d70cc4ac393158eb34311c9a2, got %s", validators[4].PublicKey.String())
	}

	if value, _ := hex.DecodeString("020000000000000000000000000000000000000000000000000000000000dEaD"); !bytes.Equal(validators[4].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 4 to have withdrawal credentials 0x020000000000000000000000000000000000000000000000000000000000dEaD, got %s", validators[4].WithdrawalCredentials)
	}

	if validators[4].Balance == nil || *validators[4].Balance != 64000000000 {
		t.Fatalf("expected validator 4 to have balance 64000000000, got %d", validators[4].Balance)
	}
}

func TestLoadValidatorsFromFile_InvalidFile(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, ``)

	_, err := LoadValidatorsFromFile(validatorsFile + "invalid")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("expected error to contain 'no such file or directory', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidPubkeyHex(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3b_not_hex:001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf:32000000000
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "encoding/hex: invalid") {
		t.Fatalf("expected error to contain 'encoding/hex: invalid', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidPubkeyLength(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b:001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf:32000000000
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid length") {
		t.Fatalf("expected error to contain 'invalid length', got %s", err)
	}
}

func TestLoadValidatorsFromFile_DuplicatePubkey(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:008aa7b9c37bf27e7c49a3185a3e721c7a02c94da7a0b6ad5f88f1b0477d3b88
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate pubkey") {
		t.Fatalf("expected error to contain 'duplicate pubkey', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidCredHex(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:0x9824e447621e4b3b_not_hex
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "encoding/hex: invalid") {
		t.Fatalf("expected error to contain 'encoding/hex: invalid', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidCredLength(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:001547805ff0547da9e51a7463a6a0c603ee
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid length") {
		t.Fatalf("expected error to contain 'invalid length', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidCredType(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:ff1547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid type") {
		t.Fatalf("expected error to contain 'invalid type', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidAddress(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:021547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid 0x01/0x02 cred") {
		t.Fatalf("expected error to contain 'invalid 0x01/0x02 cred', got %s", err)
	}
}

func TestLoadValidatorsFromFile_InvalidBalance(t *testing.T) {
	validatorsFile := createTestValidatorsFile(t, `
# <validator pubkey>:<withdrawal credentials>[:<balance>]
0x9824e447621e4b3bca7794b91c664cc0b43322a70b1881b2f804e3a990a3965a64bfe7f098cb4c0396cd0c89218de0b4:001547805ff0547da9e51a7463a6a0c603eeda01dd930f7016185f0642b9ecaf:not_a_number
`)

	_, err := LoadValidatorsFromFile(validatorsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid syntax") {
		t.Fatalf("expected error to contain 'invalid syntax', got %s", err)
	}
}
