package beaconvalidators

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	e2util "github.com/wealdtech/go-eth2-util"
)

func GenerateValidatorsByMnemonic(mnemonicsConfigPath string) ([]*Validator, error) {
	mnemonics, err := loadMnemonics(mnemonicsConfigPath)
	if err != nil {
		return nil, err
	}

	var valCount uint64

	for _, mnemonicSrc := range mnemonics {
		valCount += mnemonicSrc.Count
	}

	validators := make([]*Validator, valCount)
	offset := uint64(0)

	for m, mnemonicSrc := range mnemonics {
		var g errgroup.Group

		g.SetLimit(10_000) // when generating large states, do squeeze processing, but do not go out of memory

		var prog int32

		logrus.Infof("processing mnemonic %d, for %d validators", m, mnemonicSrc.Count)

		seed, err := seedFromMnemonic(mnemonicSrc.Mnemonic)
		if err != nil {
			return nil, fmt.Errorf("mnemonic %d is bad", m)
		}

		for i := uint64(0); i < mnemonicSrc.Count; i++ {
			valIndex := offset + i
			idx := mnemonicSrc.Start + i

			g.Go(func() error {
				signingSK, err := e2util.PrivateKeyFromSeedAndPath(seed, validatorKeyName(idx))
				if err != nil {
					return err
				}

				data := &Validator{
					PublicKey:             phase0.BLSPubKey(signingSK.PublicKey().Marshal()),
					WithdrawalCredentials: make([]byte, 32),
				}

				if mnemonicSrc.WdPrefix != "" && mnemonicSrc.WdPrefix != "0x00" && mnemonicSrc.WdAddress != "" {
					// set withdrawal address (0x01 or 0x02 credentials)
					address, err := hex.DecodeString(strings.ReplaceAll(mnemonicSrc.WdAddress, "0x", ""))
					if err != nil {
						return fmt.Errorf("failed to decode withdrawal address: %w", err)
					}

					copy(data.WithdrawalCredentials[12:], address)
					data.WithdrawalCredentials[0] = 0x01
				} else {
					// set withdrawal BLS pubkey (0x00 credentials)
					wdkeyPath := mnemonicSrc.WdKeyPath
					if wdkeyPath == "" {
						wdkeyPath = withdrawalKeyName(idx)
					}

					withdrawSK, err := e2util.PrivateKeyFromSeedAndPath(seed, wdkeyPath)
					if err != nil {
						return err
					}

					withdrawPub := withdrawSK.PublicKey().Marshal()
					h := sha256.New()
					h.Write(withdrawPub)
					copy(data.WithdrawalCredentials, h.Sum(nil))
					data.WithdrawalCredentials[0] = 0x00
				}

				if mnemonicSrc.WdPrefix != "" {
					prefix, err := hex.DecodeString(strings.ReplaceAll(mnemonicSrc.WdPrefix, "0x", ""))
					if err != nil {
						return fmt.Errorf("failed to decode withdrawal prefix: %w", err)
					}

					copy(data.WithdrawalCredentials, prefix)
				}

				// Max effective balance by default for activation
				if mnemonicSrc.Balance > 0 {
					data.Balance = &mnemonicSrc.Balance
				}

				validators[valIndex] = data
				count := atomic.AddInt32(&prog, 1)

				if count%100 == 0 {
					logrus.Infof("...validator %d/%d", count, mnemonicSrc.Count)
				}

				return nil
			})
		}

		offset += mnemonicSrc.Count

		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	return validators, nil
}

func validatorKeyName(i uint64) string {
	return fmt.Sprintf("m/12381/3600/%d/0/0", i)
}

func withdrawalKeyName(i uint64) string {
	return fmt.Sprintf("m/12381/3600/%d/0", i)
}

func seedFromMnemonic(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}

type MnemonicSrc struct {
	Mnemonic  string `yaml:"mnemonic"`
	Start     uint64 `yaml:"start"`
	Count     uint64 `yaml:"count"`
	Balance   uint64 `yaml:"balance"`
	WdAddress string `yaml:"wd_address"`
	WdPrefix  string `yaml:"wd_prefix"`
	WdKeyPath string `yaml:"wd_key_path"`
}

func loadMnemonics(srcPath string) ([]MnemonicSrc, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	var data []MnemonicSrc

	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
