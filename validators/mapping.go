package validators

import (
	"fmt"
	"os"
	"strings"
)

// MappingEntry describes a contiguous range of validators in the final state
// that originate from a single contiguous range of key indices within one
// source. It is the in-memory representation of a single validator-mapping line.
type MappingEntry struct {
	// StateIndexFrom and StateIndexTo are the inclusive validator index range in
	// the final genesis state.
	StateIndexFrom uint64
	StateIndexTo   uint64

	// Source identifies where the keys originate (e.g. "mnemonic-0").
	Source string

	// KeyIndexFrom and KeyIndexTo are the inclusive key index range within the
	// source.
	KeyIndexFrom uint64
	KeyIndexTo   uint64
}

// BuildMapping derives the validator mapping from the final, ordered validator
// list. Consecutive validators that share a source and have contiguous key
// indices are collapsed into a single entry, so a sequential (un-shuffled) set
// produces one entry per source and a block-shuffled set produces one entry per
// contiguous block.
func BuildMapping(vals []*Validator) []MappingEntry {
	entries := make([]MappingEntry, 0)
	if len(vals) == 0 {
		return entries
	}

	start := 0

	for i := 1; i <= len(vals); i++ {
		// A run continues while the next validator shares the source and its key
		// index follows the previous one. The final iteration (i == len) always
		// closes the open run.
		continues := i < len(vals) &&
			vals[i].Source == vals[start].Source &&
			vals[i].SourceKeyIndex == vals[i-1].SourceKeyIndex+1

		if continues {
			continue
		}

		//nolint:gosec // start and i-1 are loop indices, always >= 0
		entries = append(entries, MappingEntry{
			StateIndexFrom: uint64(start),
			StateIndexTo:   uint64(i - 1),
			Source:         vals[start].Source,
			KeyIndexFrom:   vals[start].SourceKeyIndex,
			KeyIndexTo:     vals[i-1].SourceKeyIndex,
		})

		start = i
	}

	return entries
}

// WriteMappingFile writes the validator mapping to path as a YAML list, one
// entry per line, in the form:
//
//   - <state-from>-<state-to>: { src: "<source>", from: <key-from>, to: <key-to> }
func WriteMappingFile(path string, vals []*Validator) error {
	entries := BuildMapping(vals)

	var sb strings.Builder

	for _, e := range entries {
		fmt.Fprintf(&sb, "- %d-%d: { src: %q, from: %d, to: %d }\n",
			e.StateIndexFrom, e.StateIndexTo, e.Source, e.KeyIndexFrom, e.KeyIndexTo)
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil { //nolint:gosec // no strict permissions needed
		return fmt.Errorf("failed to write validator mapping file: %w", err)
	}

	return nil
}
