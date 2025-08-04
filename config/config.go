package config

import (
	"encoding/binary"
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

	values := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	for key, val := range values {
		switch value := val.(type) {
		case int:
			if strings.HasSuffix(key, "_FORK_VERSION") {
				// convert to big endian byte array
				bytes := make([]byte, 4)
				binary.BigEndian.PutUint32(bytes, uint32(value)) //nolint:gosec // ignore overflow
				config.values[key] = bytes
			} else {
				config.values[key] = uint64(value) //nolint:gosec // ignore overflow
			}
		case uint64:
			config.values[key] = value
		case string:
			if strings.HasPrefix(value, "0x") {
				bytes, err := hex.DecodeString(strings.ReplaceAll(value, "0x", ""))
				if err != nil {
					return nil, fmt.Errorf("decoding hex: %w", err)
				}

				config.values[key] = bytes
			} else if val, err := strconv.ParseUint(value, 10, 64); err == nil {
				config.values[key] = val
			} else {
				config.values[key] = value
			}
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

	presetMap := make(map[string]string)
	if err := yaml.Unmarshal(presetData, &presetMap); err != nil {
		return nil, fmt.Errorf("failed to parse preset yaml: %w", err)
	}

	for key, value := range presetMap {
		if strings.HasPrefix(value, "0x") {
			bytes, err := hex.DecodeString(strings.ReplaceAll(value, "0x", ""))
			if err != nil {
				return nil, fmt.Errorf("decoding hex: %w", err)
			}

			config.preset[key] = bytes
		} else if val, err := strconv.ParseUint(value, 10, 64); err == nil {
			config.preset[key] = val
		} else {
			config.preset[key] = value
		}
	}

	return config, nil
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

	if bytes, ok := value.([]byte); ok {
		return bytes, true
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
