package beaconvalidators

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	hbls "github.com/herumi/bls-eth-go-binary/bls"
)

func createTestMnemonicsFile(t *testing.T, data string) string {
	t.Helper()
	dir := t.TempDir()

	mnemonicsFile, err := os.Create(filepath.Join(dir, "mnemonics.yaml"))
	if err != nil {
		t.Fatalf("failed to create mnemonics file: %v", err)
	}

	_, err = mnemonicsFile.WriteString(data)
	if err != nil {
		t.Fatalf("failed to write mnemonics data: %v", err)
	}

	return mnemonicsFile.Name()
}

func TestGenerateValidatorsByMnemonic_Valid(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0
  count: 1
  balance: 32000000000
  wd_prefix: "0x00"
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 1
  count: 1
  balance: 64000000000
  wd_prefix: "0x01"
  wd_address: "0x1234567890abcdef1234567890abcdef12345678"
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 10
  count: 1
  wd_prefix: "0x02"
  wd_address: "0x1234567890abcdef1234567890abcdef12345678"
`)

	err := hbls.Init(hbls.BLS12_381)
	if err != nil {
		t.Fatalf("failed to initialize BLS12-381: %v", err)
	}

	err = hbls.SetETHmode(hbls.EthModeLatest)
	if err != nil {
		t.Fatalf("failed to set ETH mode: %v", err)
	}

	validators, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err != nil {
		t.Fatalf("failed to load validators from mnemonics: %v", err)
	}

	if len(validators) != 3 {
		t.Fatalf("expected 3 validators, got %d", len(validators))
	}

	// Validator 0
	if value, _ := hex.DecodeString("a72ce460a5ab6bea347e59b17ee349bebf6adfa0a240993ed70a5be0da9638b6e2dc7bbdd19e24a8292c1c7b30f23c9e"); !bytes.Equal(validators[0].PublicKey[:], value) {
		t.Fatalf("expected validator 0 to have pubkey 0xa72ce460a5ab6bea347e59b17ee349bebf6adfa0a240993ed70a5be0da9638b6e2dc7bbdd19e24a8292c1c7b30f23c9e, got %s", validators[0].PublicKey.String())
	}

	if value, _ := hex.DecodeString("00844164a875d32ab3dd1388fb80f3376542726289c4d0a3d4270783b415b9d2"); !bytes.Equal(validators[0].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 0 to have withdrawal credentials 0x00844164a875d32ab3dd1388fb80f3376542726289c4d0a3d4270783b415b9d2, got 0x%x", validators[0].WithdrawalCredentials)
	}

	if validators[0].Balance == nil || *validators[0].Balance != 32000000000 {
		t.Fatalf("expected validator 0 to have balance 32000000000, got %d", validators[0].Balance)
	}

	// Validator 1
	if value, _ := hex.DecodeString("95300f69c73a64191af69b572724d3da8fa1dd62a0f9db32c2290ef358c2ab93006a50006d7fadffd8de583109a4446e"); !bytes.Equal(validators[1].PublicKey[:], value) {
		t.Fatalf("expected validator 1 to have pubkey 0x95300f69c73a64191af69b572724d3da8fa1dd62a0f9db32c2290ef358c2ab93006a50006d7fadffd8de583109a4446e, got %s", validators[1].PublicKey.String())
	}

	if value, _ := hex.DecodeString("0100000000000000000000001234567890abcdef1234567890abcdef12345678"); !bytes.Equal(validators[1].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 1 to have withdrawal credentials 0x0100000000000000000000001234567890abcdef1234567890abcdef12345678, got 0x%x", validators[1].WithdrawalCredentials)
	}

	if validators[1].Balance == nil || *validators[1].Balance != 64000000000 {
		t.Fatalf("expected validator 1 to have balance 64000000000, got %d", validators[1].Balance)
	}

	// Validator 2
	if value, _ := hex.DecodeString("81d086791ed8538f023575b7af4cffbbf1cfa3cf017bab1aa8fb50a858a1554b269a169d9124953046b28fd5da0353aa"); !bytes.Equal(validators[2].PublicKey[:], value) {
		t.Fatalf("expected validator 2 to have pubkey 0x81d086791ed8538f023575b7af4cffbbf1cfa3cf017bab1aa8fb50a858a1554b269a169d9124953046b28fd5da0353aa, got %s", validators[2].PublicKey.String())
	}

	if value, _ := hex.DecodeString("0200000000000000000000001234567890abcdef1234567890abcdef12345678"); !bytes.Equal(validators[2].WithdrawalCredentials, value) {
		t.Fatalf("expected validator 2 to have withdrawal credentials 0x0200000000000000000000001234567890abcdef1234567890abcdef12345678, got %s", validators[2].WithdrawalCredentials)
	}

	if validators[2].Balance != nil {
		t.Fatalf("expected validator 2 to have no balance, got %d", validators[2].Balance)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidFile(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, ``)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile + "invalid")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("expected error to contain 'no such file or directory', got %s", err)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidYaml(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0  _not yaml_
  count: 1
  balance: 32000000000
  wd_prefix: "0x00"
`)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "unmarshal errors:") {
		t.Fatalf("expected error to contain 'unmarshal errors:', got %s", err)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidMnemonic(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe invalid_word place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0
  count: 1
  balance: 32000000000
  wd_prefix: "0x00"
`)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "mnemonic 0 is bad") {
		t.Fatalf("expected error to contain 'mnemonic 0 is bad', got %s", err)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidWdAddress(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0
  count: 1
  balance: 32000000000
  wd_prefix: "0x01"
  wd_address: "invalid_address"
`)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to decode withdrawal address") {
		t.Fatalf("expected error to contain 'failed to decode withdrawal address', got %s", err)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidWdPrefix(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0
  count: 1
  balance: 32000000000
  wd_prefix: "invalid_prefix"
  wd_address: "0x1234567890abcdef1234567890abcdef12345678"
`)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to decode withdrawal prefix") {
		t.Fatalf("expected error to contain 'failed to decode withdrawal prefix', got %s", err)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidKeyIndex(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 9223372036854775807
  count: 2
  balance: 32000000000
  wd_prefix: "0x00"
  wd_address: "0x1234567890abcdef1234567890abcdef12345678"
`)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid index") {
		t.Fatalf("expected error to contain 'invalid index', got %s", err)
	}
}

func TestGenerateValidatorsByMnemonic_InvalidWdKeyIndex(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0
  count: 2
  balance: 32000000000
  wd_prefix: "0x00"
  wd_key_path: "invalid_key_path"
`)

	_, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGenerateValidatorsByMnemonic_MassKeys(t *testing.T) {
	mnemonicsFile := createTestMnemonicsFile(t, `
- mnemonic: "rare observe fox place unfold bargain cannon direct title sorry rabbit juice body autumn quality decrease mixture transfer crisp unveil path depend brick scissors"
  start: 0
  count: 200
  balance: 32000000000
  wd_prefix: "0x00"
  wd_address: "0x1234567890abcdef1234567890abcdef12345678"
`)

	validators, err := GenerateValidatorsByMnemonic(mnemonicsFile)
	if err != nil {
		t.Fatalf("expected no error, got %s", err)
	}

	if len(validators) != 200 {
		t.Fatalf("expected 200 validators, got %d", len(validators))
	}
}
