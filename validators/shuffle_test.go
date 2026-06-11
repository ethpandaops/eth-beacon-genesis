package validators

import (
	"fmt"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// makeValidators builds count validators tagged with a single source and
// sequential key indices, so tests can track where each ends up after shuffling.
func makeValidators(source string, count int) []*Validator {
	vals := make([]*Validator, count)

	var keyIndex uint64

	for i := range vals {
		var pubkey phase0.BLSPubKey
		// Encode the key index into the pubkey so order can be asserted.
		pubkey[0] = byte(keyIndex)
		pubkey[1] = byte(keyIndex >> 8)

		vals[i] = &Validator{
			PublicKey:      pubkey,
			Source:         source,
			SourceKeyIndex: keyIndex,
		}

		keyIndex++
	}

	return vals
}

func TestSeedFromForkVersion(t *testing.T) {
	tests := []struct {
		name        string
		forkVersion []byte
		want        uint64
	}{
		{name: "empty", forkVersion: nil, want: DefaultShuffleSeed},
		{name: "four-bytes", forkVersion: []byte{0x10, 0x00, 0x00, 0x38}, want: 0x10000038},
		{name: "eight-bytes", forkVersion: []byte{1, 2, 3, 4, 5, 6, 7, 8}, want: 0x0102030405060708},
		{name: "over-eight-bytes", forkVersion: []byte{0xff, 1, 2, 3, 4, 5, 6, 7, 8}, want: 0x0102030405060708},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SeedFromForkVersion(tt.forkVersion); got != tt.want {
				t.Fatalf("SeedFromForkVersion(%x) = %#x, want %#x", tt.forkVersion, got, tt.want)
			}
		})
	}
}

func TestBlockSizeForCount(t *testing.T) {
	tests := []struct {
		count int
		want  int
	}{
		{count: 0, want: minBlockSize},
		{count: 100, want: minBlockSize},
		{count: 2000, want: minBlockSize}, // 2000/100 = 20 = min
		{count: 5000, want: 50},           // 5000/100 = 50
		{count: 10000, want: maxBlockSize},
		{count: 1_000_000, want: maxBlockSize},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("count-%d", tt.count), func(t *testing.T) {
			if got := blockSizeForCount(tt.count); got != tt.want {
				t.Fatalf("blockSizeForCount(%d) = %d, want %d", tt.count, got, tt.want)
			}
		})
	}
}

func TestShuffleValidators_Deterministic(t *testing.T) {
	a := makeValidators("mnemonic-0", 1000)
	b := makeValidators("mnemonic-0", 1000)

	ShuffleValidators(a, 12345)
	ShuffleValidators(b, 12345)

	for i := range a {
		if a[i].SourceKeyIndex != b[i].SourceKeyIndex {
			t.Fatalf("same seed produced different order at index %d: %d != %d",
				i, a[i].SourceKeyIndex, b[i].SourceKeyIndex)
		}
	}
}

func TestShuffleValidators_DifferentSeed(t *testing.T) {
	a := makeValidators("mnemonic-0", 1000)
	b := makeValidators("mnemonic-0", 1000)

	ShuffleValidators(a, 1)
	ShuffleValidators(b, 2)

	identical := true

	for i := range a {
		if a[i].SourceKeyIndex != b[i].SourceKeyIndex {
			identical = false
			break
		}
	}

	if identical {
		t.Fatal("different seeds produced the same order")
	}
}

func TestShuffleValidators_PreservesSetAndChangesOrder(t *testing.T) {
	const count = 1000

	vals := makeValidators("mnemonic-0", count)
	ShuffleValidators(vals, 999)

	if len(vals) != count {
		t.Fatalf("expected %d validators after shuffle, got %d", count, len(vals))
	}

	seen := make(map[uint64]bool, count)
	ordered := true

	var idx uint64

	for _, v := range vals {
		if seen[v.SourceKeyIndex] {
			t.Fatalf("duplicate key index %d after shuffle", v.SourceKeyIndex)
		}

		seen[v.SourceKeyIndex] = true

		if v.SourceKeyIndex != idx {
			ordered = false
		}

		idx++
	}

	if len(seen) != count {
		t.Fatalf("expected %d unique validators, got %d", count, len(seen))
	}

	if ordered {
		t.Fatal("shuffle did not change the ordering")
	}
}

func TestShuffleValidators_PreservesWithinBlockOrder(t *testing.T) {
	// With 1000 validators the block size is minBlockSize (20). Within any
	// shuffled block the key indices must remain strictly increasing by one.
	vals := makeValidators("mnemonic-0", 1000)
	ShuffleValidators(vals, 7)

	blockSize := blockSizeForCount(1000)
	for start := 0; start < len(vals); start += blockSize {
		for i := start + 1; i < start+blockSize && i < len(vals); i++ {
			if vals[i].SourceKeyIndex != vals[i-1].SourceKeyIndex+1 {
				t.Fatalf("within-block order broken at index %d: %d after %d",
					i, vals[i].SourceKeyIndex, vals[i-1].SourceKeyIndex)
			}
		}
	}
}

func TestShuffleValidators_SmallSetNoOp(t *testing.T) {
	vals := makeValidators("mnemonic-0", minBlockSize)
	ShuffleValidators(vals, 42)

	var idx uint64

	for _, v := range vals {
		if v.SourceKeyIndex != idx {
			t.Fatalf("small set should not be shuffled, but index %d moved", idx)
		}

		idx++
	}
}
