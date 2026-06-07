package validators

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildMapping_Sequential(t *testing.T) {
	// Two sources concatenated sequentially collapse into one entry each.
	vals := append(makeValidators("mnemonic-0", 50), makeValidators("additional-validators", 30)...)

	entries := BuildMapping(vals)

	if len(entries) != 2 {
		t.Fatalf("expected 2 mapping entries, got %d", len(entries))
	}

	want := []MappingEntry{
		{StateIndexFrom: 0, StateIndexTo: 49, Source: "mnemonic-0", KeyIndexFrom: 0, KeyIndexTo: 49},
		{StateIndexFrom: 50, StateIndexTo: 79, Source: "additional-validators", KeyIndexFrom: 0, KeyIndexTo: 29},
	}

	for i, w := range want {
		if entries[i] != w {
			t.Fatalf("entry %d = %+v, want %+v", i, entries[i], w)
		}
	}
}

func TestBuildMapping_Empty(t *testing.T) {
	if entries := BuildMapping(nil); len(entries) != 0 {
		t.Fatalf("expected no entries for empty input, got %d", len(entries))
	}
}

func TestBuildMapping_RangesCoverAllAndStayContiguous(t *testing.T) {
	const count = 1000

	vals := makeValidators("mnemonic-0", count)
	ShuffleValidators(vals, 555)

	entries := BuildMapping(vals)
	if len(entries) < 2 {
		t.Fatalf("expected the shuffle to produce multiple entries, got %d", len(entries))
	}

	var covered uint64

	for i, e := range entries {
		// State indices must be contiguous and cover the whole set.
		if e.StateIndexFrom != covered {
			t.Fatalf("entry %d state range starts at %d, expected %d", i, e.StateIndexFrom, covered)
		}

		stateLen := e.StateIndexTo - e.StateIndexFrom
		keyLen := e.KeyIndexTo - e.KeyIndexFrom

		// Each entry maps an equal-length state and key range (contiguous block).
		if stateLen != keyLen {
			t.Fatalf("entry %d state length %d != key length %d", i, stateLen, keyLen)
		}

		// The validator at the start of each state range must carry the entry's
		// source key index.
		if vals[e.StateIndexFrom].SourceKeyIndex != e.KeyIndexFrom {
			t.Fatalf("entry %d key-from %d does not match validator at state %d (%d)",
				i, e.KeyIndexFrom, e.StateIndexFrom, vals[e.StateIndexFrom].SourceKeyIndex)
		}

		covered = e.StateIndexTo + 1
	}

	if covered != count {
		t.Fatalf("entries cover %d validators, expected %d", covered, count)
	}
}

func TestWriteMappingFile(t *testing.T) {
	vals := append(makeValidators("mnemonic-0", 20), makeValidators("additional-validators", 20)...)

	path := filepath.Join(t.TempDir(), "mapping.yaml")
	if err := WriteMappingFile(path, vals); err != nil {
		t.Fatalf("WriteMappingFile failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read mapping file: %v", err)
	}

	want := "- 0-19: { src: \"mnemonic-0\", from: 0, to: 19 }\n" +
		"- 20-39: { src: \"additional-validators\", from: 0, to: 19 }\n"

	if string(data) != want {
		t.Fatalf("unexpected mapping file content:\ngot:\n%s\nwant:\n%s", string(data), want)
	}
}
