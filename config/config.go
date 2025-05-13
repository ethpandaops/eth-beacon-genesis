package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/eth-beacon-genesis/config/presets"
)

type Config struct {
	values map[string]interface{}
	preset map[string]interface{}
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{
		values: make(map[string]interface{}),
		preset: make(map[string]interface{}),
	}

	// load config from yaml
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// First try to parse with a more flexible structure
	var rawValues map[string]interface{}
	if err := yaml.Unmarshal(data, &rawValues); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	// Process the values, ignoring complex structures
	for key, value := range rawValues {
		// Skip complex structures entirely
		if isComplexStructure(value) {
			continue
		}

		// Handle string values
		if strValue, ok := value.(string); ok {
			// Special handling for fork version fields which are always hex
			if strings.HasSuffix(key, "_FORK_VERSION") && strings.HasPrefix(strValue, "0x") {
				bytes, err := hex.DecodeString(strings.ReplaceAll(strValue, "0x", ""))
				if err != nil {
					return nil, fmt.Errorf("decoding hex for %s: %w", key, err)
				}

				config.values[key] = bytes
			} else if strings.HasPrefix(strValue, "0x") {
				bytes, err := hex.DecodeString(strings.ReplaceAll(strValue, "0x", ""))
				if err != nil {
					return nil, fmt.Errorf("decoding hex: %w", err)
				}

				config.values[key] = bytes
			} else if val, err := strconv.ParseUint(strValue, 10, 64); err == nil {
				config.values[key] = val
			} else {
				config.values[key] = strValue
			}
		} else {
			// For other scalar types, store as-is
			config.values[key] = value
		}
	}

	// load referenced preset
	presetName, found := config.GetString("PRESET_BASE")
	if !found || presetName == "" {
		return nil, fmt.Errorf("preset not found")
	}

	presetData, err := presets.PresetsFS.ReadFile(presetName + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("preset '%v' not found: %w", presetName, err)
	}

	// Use the same approach for presets
	var rawPresets map[string]interface{}
	if err := yaml.Unmarshal(presetData, &rawPresets); err != nil {
		return nil, fmt.Errorf("failed to parse preset yaml: %w", err)
	}

	for key, value := range rawPresets {
		// Skip complex structures entirely
		if isComplexStructure(value) {
			continue
		}

		if strValue, ok := value.(string); ok {
			// Special handling for fork version fields which are always hex
			if strings.HasSuffix(key, "_FORK_VERSION") && strings.HasPrefix(strValue, "0x") {
				bytes, err := hex.DecodeString(strings.ReplaceAll(strValue, "0x", ""))
				if err != nil {
					return nil, fmt.Errorf("decoding hex for %s: %w", key, err)
				}

				config.preset[key] = bytes
			} else if strings.HasPrefix(strValue, "0x") {
				bytes, err := hex.DecodeString(strings.ReplaceAll(strValue, "0x", ""))
				if err != nil {
					return nil, fmt.Errorf("decoding hex: %w", err)
				}

				config.preset[key] = bytes
			} else if val, err := strconv.ParseUint(strValue, 10, 64); err == nil {
				config.preset[key] = val
			} else {
				config.preset[key] = strValue
			}
		} else {
			config.preset[key] = value
		}
	}

	return config, nil
}

// Helper function to check if a value is a complex structure (not a simple scalar)
func isComplexStructure(value interface{}) bool {
	switch value.(type) {
	case map[string]interface{}, []interface{}:
		return true
	default:
		return false
	}
}

func (c *Config) Get(key string) (interface{}, bool) {
	value, ok := c.values[key]

	if !ok {
		value, ok = c.preset[key]
	}

	return value, ok
}

func (c *Config) GetString(key string) (string, bool) {
	value, ok := c.Get(key)
	if !ok {
		return "", false
	}

	if str, ok := value.(string); ok {
		return str, true
	}

	return "", false
}

func (c *Config) GetUint(key string) (uint64, bool) {
	value, ok := c.Get(key)
	if !ok {
		return 0, false
	}

	if val, ok := value.(uint64); ok {
		return val, true
	}

	return 0, false
}

func (c *Config) GetUintDefault(key string, defaultVal uint64) uint64 {
	value, ok := c.GetUint(key)
	if !ok {
		return defaultVal
	}

	return value
}

func (c *Config) GetBytes(key string) ([]byte, bool) {
	value, ok := c.Get(key)
	if !ok {
		return nil, false
	}

	// If it's already a byte slice, return it
	if bytes, ok := value.([]byte); ok {
		return bytes, true
	}

	// If it's a string, try to convert it to bytes if it's a hex string
	if str, ok := value.(string); ok && strings.HasPrefix(str, "0x") {
		bytes, err := hex.DecodeString(strings.ReplaceAll(str, "0x", ""))
		if err == nil {
			return bytes, true
		}
	}

	return nil, false
}

func (c *Config) GetBytesDefault(key string, defaultVal []byte) []byte {
	value, ok := c.GetBytes(key)
	if !ok {
		return defaultVal
	}

	return value
}

func (c *Config) GetSpecs() map[string]interface{} {
	specs := make(map[string]interface{})

	for k, v := range c.preset {
		specs[k] = v
	}

	for k, v := range c.values {
		specs[k] = v
	}

	return specs
}
