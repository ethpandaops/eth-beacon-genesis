package leanvalidators

import (
	"fmt"
	"net"
	"os"
	"encoding/hex"
	"strconv"

	"github.com/ethpandaops/eth-beacon-genesis/leanutils"
	"github.com/ethereum/go-ethereum/crypto"
	"gopkg.in/yaml.v3"
)

type ShuffleMode string

const (
	ShuffleModeNone       ShuffleMode = "none"
	ShuffleModeRoundRobin ShuffleMode = "roundrobin"
)

type ENRFields struct {
	IP   string            `yaml:"ip,omitempty"`
	IP6  string            `yaml:"ip6,omitempty"`
	TCP  int               `yaml:"tcp,omitempty"`
	UDP  int               `yaml:"udp,omitempty"`
	QUIC int               `yaml:"quic,omitempty"`
	Seq  uint64            `yaml:"seq,omitempty"`
	Custom map[string]string `yaml:",inline"`
}

type MassValidatorEntry struct {
	// Validator name prefix for generated validators
	Name string `yaml:"name,omitempty"`
	
	// Legacy field - use ENR string directly
	ENR string `yaml:"enr,omitempty"`
	
	// New fields - generate ENR from privkey and fields
	PrivKey   string    `yaml:"privkey,omitempty"`
	ENRFields ENRFields `yaml:"enrFields,omitempty"`
	
	Count int `yaml:"count"`
}

type MassValidatorsConfig struct {
	Shuffle    ShuffleMode          `yaml:"shuffle"`
	Validators []MassValidatorEntry `yaml:"validators"`
}

func LoadValidatorsFromMassYaml(validatorsConfigPath string) ([]*Validator, error) {
	validatorsFile, err := os.ReadFile(validatorsConfigPath)
	if err != nil {
		return nil, err
	}

	config := &MassValidatorsConfig{
		Shuffle: ShuffleModeNone, // default
	}

	err = yaml.Unmarshal(validatorsFile, config)
	if err != nil {
		return nil, err
	}

	return expandValidators(config)
}

func expandValidators(config *MassValidatorsConfig) ([]*Validator, error) {
	if len(config.Validators) == 0 {
		return nil, fmt.Errorf("no validators defined in configuration")
	}

	// Generate ENRs for entries that use privkey+fields
	processedEntries := make([]processedEntry, len(config.Validators))
	for i, entry := range config.Validators {
		enrString, err := generateENRFromEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ENR for entry %d: %w", i, err)
		}
		processedEntries[i] = processedEntry{
			Name:  entry.Name,
			ENR:   enrString,
			Count: entry.Count,
		}
	}

	// Calculate total validators
	totalValidators := 0
	for _, entry := range processedEntries {
		if entry.Count < 0 {
			return nil, fmt.Errorf("invalid count %d for ENR %s", entry.Count, entry.ENR)
		}
		totalValidators += entry.Count
	}

	if totalValidators == 0 {
		return nil, fmt.Errorf("no validators to generate (all counts are 0)")
	}

	validators := make([]*Validator, 0, totalValidators)

	switch config.Shuffle {
	case ShuffleModeNone, "":
		// Linear mode: add all validators for each ENR in order
		for _, entry := range processedEntries {
			for i := 0; i < entry.Count; i++ {
				validators = append(validators, &Validator{
					Name: generateValidatorName(entry.Name, i),
					ENR:  entry.ENR,
				})
			}
		}

	case ShuffleModeRoundRobin:
		// Round-robin mode: distribute validators evenly
		counters := make([]int, len(processedEntries))
		remaining := totalValidators

		for remaining > 0 {
			for idx, entry := range processedEntries {
				if counters[idx] < entry.Count {
					validators = append(validators, &Validator{
						Name: generateValidatorName(entry.Name, counters[idx]),
						ENR:  entry.ENR,
					})
					counters[idx]++
					remaining--

					if remaining == 0 {
						break
					}
				}
			}
		}

	default:
		return nil, fmt.Errorf("unknown shuffle mode: %s", config.Shuffle)
	}

	return validators, nil
}

type processedEntry struct {
	Name  string
	ENR   string
	Count int
}

func generateENRFromEntry(entry MassValidatorEntry) (string, error) {
	// If ENR is provided directly, use it
	if entry.ENR != "" {
		return entry.ENR, nil
	}

	// If privkey is provided, generate ENR from fields
	if entry.PrivKey != "" {
		return generateENRFromPrivKeyAndFields(entry.PrivKey, entry.ENRFields)
	}

	return "", fmt.Errorf("either enr or privkey must be specified")
}

func generateENRFromPrivKeyAndFields(privKeyHex string, fields ENRFields) (string, error) {
	// Parse private key
	privKey, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}

	// Create new ENR
	enrObj, err := leanutils.NewENR()
	if err != nil {
		return "", fmt.Errorf("failed to create ENR: %w", err)
	}

	// Set standard fields
	if fields.IP != "" {
		ip := net.ParseIP(fields.IP)
		if ip == nil {
			return "", fmt.Errorf("invalid IP address: %s", fields.IP)
		}
		enrObj.SetIP4(ip)
	}

	if fields.IP6 != "" {
		ip6 := net.ParseIP(fields.IP6)
		if ip6 == nil {
			return "", fmt.Errorf("invalid IPv6 address: %s", fields.IP6)
		}
		enrObj.SetIP6(ip6)
	}

	if fields.TCP > 0 {
		enrObj.SetTCP(fields.TCP)
	}

	if fields.UDP > 0 {
		enrObj.SetUDP(fields.UDP)
	}

	if fields.QUIC > 0 {
		enrObj.SetEntry("quic", uint16(fields.QUIC))
	}

	// Set custom fields
	for key, valueHex := range fields.Custom {
		// Skip known fields
		if key == "ip" || key == "ip6" || key == "tcp" || key == "udp" || key == "quic" || key == "seq" {
			continue
		}

		// Try to parse as hex bytes
		if len(valueHex) > 0 && valueHex[0:2] == "0x" {
			bytes, err := hex.DecodeString(valueHex[2:])
			if err != nil {
				return "", fmt.Errorf("invalid hex value for field %s: %w", key, err)
			}
			enrObj.SetEntry(key, bytes)
		} else {
			// Try to parse as number first, then fall back to string
			if num, err := strconv.ParseUint(valueHex, 10, 64); err == nil {
				enrObj.SetEntry(key, num)
			} else {
				enrObj.SetEntry(key, valueHex)
			}
		}
	}

	// Set sequence number
	if fields.Seq > 0 {
		enrObj.SetSeq(fields.Seq)
	} else {
		enrObj.SetSeq(1)
	}

	// Sign the ENR
	if err := enrObj.Sign(privKey); err != nil {
		return "", fmt.Errorf("failed to sign ENR: %w", err)
	}

	// Encode to string
	enrString, err := enrObj.Encode()
	if err != nil {
		return "", fmt.Errorf("failed to encode ENR: %w", err)
	}

	return enrString, nil
}

func generateValidatorName(baseName string, index int) string {
	if baseName == "" {
		return fmt.Sprintf("validator_%d", index)
	}
	// Always return the base name - don't add index suffix
	return baseName
}
