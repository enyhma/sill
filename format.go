package sill

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// FormatQueryResults renders a Query result as "json", "csv", or "table"
// (one row per matching note, by vault-relative path). sill evaluates
// filters, it doesn't render Obsidian's own column/card/map views (see
// CTX-jOPDBK9Zwu's non-goals) — this is the stable, documented contract
// CAP-KmjBkVzY5Y's acceptance criteria calls for, not a view renderer.
func FormatQueryResults(notes []*EvalContext, format string) (string, error) {
	paths := make([]string, len(notes))
	for i, n := range notes {
		paths[i] = n.File.Path
	}

	switch format {
	case "json":
		b, err := json.MarshalIndent(paths, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil

	case "csv":
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		for _, p := range paths {
			if err := w.Write([]string{p}); err != nil {
				return "", err
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return "", err
		}
		return buf.String(), nil

	case "table", "":
		return strings.Join(paths, "\n"), nil

	default:
		return "", fmt.Errorf("unknown format %q (expected json, csv, or table)", format)
	}
}
