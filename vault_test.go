package sill

import (
	"os"
	"path/filepath"
	"testing"
)

func writeNote(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadVault(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "reviewed.md", "---\nstatus: reviewed\ntags: [book, fiction]\n---\n\nSee [[Other Note]] and #urgent.\n")
	writeNote(t, dir, "sub/open.md", "---\nstatus: open\n---\n\nNothing here.\n")

	notes, err := LoadVault(dir)
	if err != nil {
		t.Fatalf("LoadVault: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("got %d notes, want 2", len(notes))
	}

	var reviewed, open *EvalContext
	for _, n := range notes {
		switch n.Note["status"] {
		case "reviewed":
			reviewed = n
		case "open":
			open = n
		}
	}
	if reviewed == nil || open == nil {
		t.Fatalf("missing expected notes: %+v", notes)
	}

	if reviewed.File.Name != "reviewed.md" || reviewed.File.Basename != "reviewed" || reviewed.File.Ext != "md" {
		t.Errorf("reviewed.File = %+v", reviewed.File)
	}
	if reviewed.File.Folder != "" {
		t.Errorf("reviewed.File.Folder = %q, want empty (vault root)", reviewed.File.Folder)
	}
	if open.File.Folder != "sub" {
		t.Errorf("open.File.Folder = %q, want %q", open.File.Folder, "sub")
	}

	wantTags := map[string]bool{"book": true, "fiction": true, "urgent": true}
	if len(reviewed.File.Tags) != len(wantTags) {
		t.Errorf("reviewed.File.Tags = %v, want %v", reviewed.File.Tags, wantTags)
	}
	for _, tag := range reviewed.File.Tags {
		if !wantTags[tag] {
			t.Errorf("unexpected tag %q", tag)
		}
	}

	if len(reviewed.File.Links) != 1 || reviewed.File.Links[0] != "Other Note" {
		t.Errorf("reviewed.File.Links = %v", reviewed.File.Links)
	}
}

func TestLoadVault_SkipsCodeFenceTag(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "n.md", "---\nstatus: open\n---\n\n```\n#notareal tag\n```\n#real\n")

	notes, err := LoadVault(dir)
	if err != nil {
		t.Fatalf("LoadVault: %v", err)
	}
	if len(notes[0].File.Tags) != 1 || notes[0].File.Tags[0] != "real" {
		t.Errorf("Tags = %v, want [real]", notes[0].File.Tags)
	}
}

func TestQuery_GlobalAndViewFilters(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "---\nstatus: reviewed\nkind: book\n---\n\nbody\n")
	writeNote(t, dir, "b.md", "---\nstatus: reviewed\nkind: article\n---\n\nbody\n")
	writeNote(t, dir, "c.md", "---\nstatus: draft\nkind: book\n---\n\nbody\n")

	base := &Base{
		Filters: &Filter{Statement: `status == "reviewed"`},
		Views: []View{
			{Name: "books", Filters: &Filter{Statement: `kind == "book"`}},
		},
	}

	results, err := Query(base, dir, "books")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) != 1 || results[0].File.Name != "a.md" {
		t.Errorf("Query results = %+v, want just a.md", results)
	}

	all, err := Query(base, dir, "")
	if err != nil {
		t.Fatalf("Query (default view): %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Query with default (first) view = %d results, want 1", len(all))
	}

	if _, err := Query(base, dir, "nonexistent"); err == nil {
		t.Error("expected an error for an unknown view name")
	}
}

func TestLint_EndToEnd(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "---\nstatus: reviewed\n---\n\nbody\n")
	writeNote(t, dir, "b.md", "---\nstatus: reviewed\n---\n\nbody\n")
	writeNote(t, dir, "c.md", "---\nstatus: draft\n---\n\nbody\n")

	content := `
sillAssert:
  population: 'status != "draft"'
  condition: 'status == "reviewed"'
  threshold: 1.0
`
	base, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	res, err := Lint(base, dir)
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if res.PopulationCount != 2 || res.SatisfiedCount != 2 || !res.Passed {
		t.Errorf("res = %+v", res)
	}

	plain, err := Parse([]byte("views:\n  - type: table\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := Lint(plain, dir); err == nil {
		t.Error("expected an error linting a base with no sillAssert extension")
	}
}

func TestFormatQueryResults(t *testing.T) {
	notes := []*EvalContext{
		{File: FileMeta{Path: "a.md"}},
		{File: FileMeta{Path: "b.md"}},
	}

	table, err := FormatQueryResults(notes, "table")
	if err != nil || table != "a.md\nb.md" {
		t.Errorf("table = %q, %v", table, err)
	}

	csv, err := FormatQueryResults(notes, "csv")
	if err != nil || csv != "a.md\nb.md\n" {
		t.Errorf("csv = %q, %v", csv, err)
	}

	jsonOut, err := FormatQueryResults(notes, "json")
	if err != nil || jsonOut != "[\n  \"a.md\",\n  \"b.md\"\n]" {
		t.Errorf("json = %q, %v", jsonOut, err)
	}

	if _, err := FormatQueryResults(notes, "bogus"); err == nil {
		t.Error("expected an error for an unknown format")
	}
}
