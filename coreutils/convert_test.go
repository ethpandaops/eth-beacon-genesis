package coreutils

import (
	"testing"
)

func TestUintToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []byte
	}{
		{
			name:     "uint8",
			input:    uint8(255),
			expected: []byte{255},
		},
		{
			name:     "uint16",
			input:    uint16(65535),
			expected: []byte{255, 255},
		},
		{
			name:     "uint32",
			input:    uint32(4294967295),
			expected: []byte{255, 255, 255, 255},
		},
		{
			name:     "uint64",
			input:    uint64(18446744073709551615),
			expected: []byte{255, 255, 255, 255, 255, 255, 255, 255},
		},
		{
			name:     "uint8 zero",
			input:    uint8(0),
			expected: []byte{0},
		},
		{
			name:     "uint16 zero",
			input:    uint16(0),
			expected: []byte{0, 0},
		},
		{
			name:     "uint32 zero",
			input:    uint32(0),
			expected: []byte{0, 0, 0, 0},
		},
		{
			name:     "uint64 zero",
			input:    uint64(0),
			expected: []byte{0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name:     "invalid type",
			input:    "invalid",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UintToBytes(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("UintToBytes() got length = %v, want %v", len(result), len(tt.expected))
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("UintToBytes() = %v, want %v", result, tt.expected)
					break
				}
			}
		})
	}
}

func TestBytesToUint(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint64
	}{
		{
			name:     "1 byte",
			input:    []byte{255},
			expected: 255,
		},
		{
			name:     "2 bytes",
			input:    []byte{255, 255},
			expected: 65535,
		},
		{
			name:     "4 bytes",
			input:    []byte{255, 255, 255, 255},
			expected: 4294967295,
		},
		{
			name:     "8 bytes",
			input:    []byte{255, 255, 255, 255, 255, 255, 255, 255},
			expected: 18446744073709551615,
		},
		{
			name:     "1 byte zero",
			input:    []byte{0},
			expected: 0,
		},
		{
			name:     "2 bytes zero",
			input:    []byte{0, 0},
			expected: 0,
		},
		{
			name:     "4 bytes zero",
			input:    []byte{0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "8 bytes zero",
			input:    []byte{0, 0, 0, 0, 0, 0, 0, 0},
			expected: 0,
		},
		{
			name:     "invalid length",
			input:    []byte{1, 2, 3},
			expected: 0,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BytesToUint(tt.input); got != tt.expected {
				t.Errorf("BytesToUint() = %v, want %v", got, tt.expected)
			}
		})
	}
}
