package sill

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/enyhma/hearth"
)

// RenderConfig controls how FormatQueryResults presents a Query result.
//
// Format is one of "table" (default), "grid", "json", "toon", or "csv".
// Table is the full-detail option: one block per note, every property,
// nested values shown through indentation (hearth.RenderBlock) — for a
// person reading a terminal. Grid/toon/csv are the compact-overview
// options: a fixed column set, nested values flattened inline
// (hearth.FormatInline) rather than indented, since none of those three
// can recurse within a cell. JSON always includes every property on
// every note (Columns doesn't apply to it) since nesting there is free.
//
// Columns selects which properties grid/toon/csv show, in order —
// typically a view's Order. Empty means "every property seen across the
// result set, sorted" (table's default scope, just column-shaped).
type RenderConfig struct {
	Format  string
	Color   bool
	Columns []string
}

// FormatQueryResults renders a Query result per cfg. See RenderConfig.
func FormatQueryResults(notes []*EvalContext, cfg RenderConfig) (string, error) {
	switch cfg.Format {
	case "", "table":
		return renderTable(notes, cfg.Color), nil
	case "grid":
		return renderGrid(notes, cfg.Columns), nil
	case "json":
		return renderJSON(notes)
	case "toon":
		return renderToon(notes, cfg.Columns), nil
	case "csv":
		return renderCSV(notes, cfg.Columns)
	default:
		return "", fmt.Errorf("unknown format %q (expected table, grid, json, toon, or csv)", cfg.Format)
	}
}

func renderTable(notes []*EvalContext, color bool) string {
	items := make([]hearth.BlockItem, len(notes))
	for i, n := range notes {
		items[i] = hearth.BlockItem{Header: n.File.Path, Props: sortedProps(n.Note)}
	}
	return hearth.RenderBlock(items, color, nil)
}

func renderGrid(notes []*EvalContext, columns []string) string {
	cols := columnsOrAll(notes, columns)
	tbl := &hearth.Table{Headers: gridHeaders(cols)}
	for _, n := range notes {
		tbl.Rows = append(tbl.Rows, gridRow(n, cols))
	}
	return tbl.Render()
}

func renderToon(notes []*EvalContext, columns []string) string {
	cols := columnsOrAll(notes, columns)
	fields := append([]string{"path"}, cols...)
	rows := make([][]string, len(notes))
	for i, n := range notes {
		rows[i] = gridRow(n, cols)
	}
	return hearth.RenderTOON("notes", fields, rows)
}

func renderCSV(notes []*EvalContext, columns []string) (string, error) {
	cols := columnsOrAll(notes, columns)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(gridHeaders(cols)); err != nil {
		return "", err
	}
	for _, n := range notes {
		if err := w.Write(gridRow(n, cols)); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderJSON(notes []*EvalContext) (string, error) {
	out := make([]map[string]any, len(notes))
	for i, n := range notes {
		m := map[string]any{"path": n.File.Path}
		for k, v := range n.Note {
			m[k] = v
		}
		out[i] = m
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// gridHeaders/gridRow build a PATH column plus cols, flattening any
// nested value inline (hearth.FormatInline) since grid/toon/csv rows
// can't recurse the way table's blocks do.
func gridHeaders(cols []string) []string {
	headers := make([]string, len(cols)+1)
	headers[0] = "path"
	for i, c := range cols {
		headers[i+1] = c
	}
	return headers
}

// gridRow resolves each column through the real expression evaluator,
// not a raw map lookup — a view's order: commonly names file.*/note.*/
// formula.* references (file.name, created, formula.ppu), not just bare
// frontmatter properties. An unresolvable column (a typo, or a formula
// that errors on this particular note) renders as an empty cell rather
// than failing the whole row.
func gridRow(n *EvalContext, cols []string) []string {
	row := make([]string, len(cols)+1)
	row[0] = n.File.Path
	for i, c := range cols {
		v, err := EvalExpr(c, n)
		if err != nil {
			row[i+1] = ""
			continue
		}
		row[i+1] = hearth.FormatInline(v)
	}
	return row
}

// columnsOrAll returns columns unchanged when non-empty, otherwise every
// property key seen across notes, sorted for determinism.
func columnsOrAll(notes []*EvalContext, columns []string) []string {
	if len(columns) > 0 {
		return columns
	}
	seen := make(map[string]bool)
	var cols []string
	for _, n := range notes {
		for k := range n.Note {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	sort.Strings(cols)
	return cols
}

// sortedProps turns a note's frontmatter map into hearth.Prop entries in
// a deterministic (alphabetical) key order — a plain Go map has none.
func sortedProps(note map[string]any) []hearth.Prop {
	keys := make([]string, 0, len(note))
	for k := range note {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	props := make([]hearth.Prop, len(keys))
	for i, k := range keys {
		props[i] = hearth.Prop{Key: k, Value: note[k]}
	}
	return props
}
