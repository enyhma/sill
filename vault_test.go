package sill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
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

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	if len(notes[0].File.Tags) != 1 || notes[0].File.Tags[0] != "real" {
		t.Errorf("Tags = %v, want [real]", notes[0].File.Tags)
	}
}

// TestLoadVault_CrossNoteResolution is the fix for a real gap: EvalContext
// used to be built per-note in isolation, so file(), link.asFile(), and
// link.linksTo() had no way to look up any OTHER note even though
// LoadVault already has the whole vault in hand via hearth.ParseDir. This
// proves the shared Vault (hearth.BuildGraph + hearth.Match under the
// hood) actually resolves across notes now.
func TestLoadVault_CrossNoteResolution(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "---\nstatus: open\n---\n\nSee [[Other Note]].\n")
	writeNote(t, dir, "Other Note.md", "---\ntags: [reviewed]\n---\n\nLinks to [[Third]].\n")
	writeNote(t, dir, "Third.md", "---\n---\n\nNo outgoing links.\n")

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	var a *EvalContext
	for _, n := range notes {
		if n.File.Name == "a.md" {
			a = n
		}
	}
	if a == nil {
		t.Fatal("a.md not found")
	}

	// file("Other Note") resolves to the other note, and methods/fields
	// on the result read ITS data, not a's.
	resolved := mustEval(t, `file("Other Note")`, a)
	fr, ok := resolved.(fileRef)
	if !ok || fr.File.Name != "Other Note.md" {
		t.Fatalf("file(\"Other Note\") = %#v", resolved)
	}
	if got := mustEval(t, `file("Other Note").hasTag("reviewed")`, a); got != true {
		t.Errorf("file(\"Other Note\").hasTag(\"reviewed\") = %v", got)
	}
	if got := mustEval(t, `file("Other Note").name`, a); got != "Other Note.md" {
		t.Errorf("file(\"Other Note\").name = %v", got)
	}

	// An unresolved path is nil, not an error (same as a missing property).
	if got, err := EvalExpr(`file("Nonexistent")`, a); err != nil || got != nil {
		t.Errorf("file(\"Nonexistent\") = %v, %v, want nil, nil", got, err)
	}

	// link(...).asFile() goes through the same resolver.
	if got := mustEval(t, `link("Other Note").asFile().hasTag("reviewed")`, a); got != true {
		t.Errorf("link(\"Other Note\").asFile().hasTag(\"reviewed\") = %v", got)
	}

	// link(...).linksTo(...) checks the LINK TARGET's own outgoing links,
	// not a's — Other Note links to Third, but not back to a.
	if got := mustEval(t, `link("Other Note").linksTo("Third")`, a); got != true {
		t.Errorf("link(\"Other Note\").linksTo(\"Third\") = %v, want true", got)
	}
	if got := mustEval(t, `link("Other Note").linksTo("a")`, a); got != false {
		t.Errorf("link(\"Other Note\").linksTo(\"a\") = %v, want false", got)
	}
}

// TestLoadVault_ToleratesBadNote mirrors the real-world case that drove
// hearth's DEC-LSh8yPXvpn: an Obsidian template file with Templater
// placeholder syntax isn't valid YAML. LoadVault must skip it and still
// return every other note, not fail the whole load.
// TestLoadVault_LinksEmbedsBacklinks covers three fields found missing
// (and one bug found) while testing against real vaults: file.links was
// conflating true [[links]] with ![[embeds]] (real Obsidian keeps them
// separate), file.embeds didn't exist at all, and file.backlinks (notes
// that link TO this one) wasn't wired to anything despite hearth's graph
// already computing it (graph.In).
func TestLoadVault_LinksEmbedsBacklinks(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "---\n---\n\nSee [[b]] and embed ![[c]].\n")
	writeNote(t, dir, "b.md", "---\n---\n\nAlso links [[a]].\n")
	writeNote(t, dir, "c.md", "---\n---\n\nNo links.\n")

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	var a, c *EvalContext
	for _, n := range notes {
		switch n.File.Name {
		case "a.md":
			a = n
		case "c.md":
			c = n
		}
	}

	if len(a.File.Links) != 1 || a.File.Links[0] != "b" {
		t.Errorf("a.File.Links = %v, want [b] (c is an embed, not a link)", a.File.Links)
	}
	if len(a.File.Embeds) != 1 || a.File.Embeds[0] != "c" {
		t.Errorf("a.File.Embeds = %v, want [c]", a.File.Embeds)
	}

	if got := mustEval(t, "file.backlinks", a); toAnySliceEq(got, []string{"b.md"}) == false {
		t.Errorf("a's backlinks = %v, want [b.md]", got)
	}
	if got := mustEval(t, "file.backlinks", c); toAnySliceEq(got, []string{"a.md"}) == false {
		t.Errorf("c's backlinks = %v, want [a.md] (an embed counts as a backlink)", got)
	}
}

func toAnySliceEq(got any, want []string) bool {
	list, ok := got.([]any)
	if !ok || len(list) != len(want) {
		return false
	}
	for i, w := range want {
		if list[i] != w {
			return false
		}
	}
	return true
}

// TestLoadVault_FrontmatterWikilinkResolves reproduces the real pattern
// from kepano-obsidian's vault — categories: ["[[Albums]]"] — and
// confirms it resolves through file.hasLink/backlinks the same way a
// body-text [[wikilink]] does.
func TestLoadVault_FrontmatterWikilinkResolves(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "---\ncategories:\n  - \"[[Albums]]\"\n---\n\nbody\n")
	writeNote(t, dir, "Albums.md", "---\n---\n\nbody\n")

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	var a *EvalContext
	for _, n := range notes {
		if n.File.Name == "a.md" {
			a = n
		}
	}
	if got := mustEval(t, `file.hasLink("Albums")`, a); got != true {
		t.Errorf(`file.hasLink("Albums") = %v, want true`, got)
	}
	if got, err := EvalExpr(`file.backlinks`, &EvalContext{File: FileMeta{Path: "Albums.md"}, Vault: a.Vault}); err != nil || !toAnySliceEq(got, []string{"a.md"}) {
		t.Errorf("Albums.md backlinks = %v, %v, want [a.md]", got, err)
	}
}

// TestLoadVault_WikilinkFragmentResolves reproduces the real embed shape
// from kepano-obsidian's Kyoto.md: ![[Trips.base#Location]] — the
// fragment must not prevent Trips.base.md itself from resolving.
func TestLoadVault_WikilinkFragmentResolves(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "Kyoto.md", "---\n---\n\n![[Trips.base#Location]]\n")
	writeNote(t, dir, "Trips.base.md", "---\n---\n\nbody\n")

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	var kyoto *EvalContext
	for _, n := range notes {
		if n.File.Name == "Kyoto.md" {
			kyoto = n
		}
	}
	if got := mustEval(t, `file("Trips.base").name`, kyoto); got != "Trips.base.md" {
		t.Errorf(`file("Trips.base").name = %v, want Trips.base.md (fragment stripped)`, got)
	}
}

// TestLoadVault_AliasResolves confirms a note is resolvable by its own
// aliases: entry, not just its filename.
func TestLoadVault_AliasResolves(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "Full Name.md", "---\naliases:\n  - Nickname\n---\n\nbody\n")
	writeNote(t, dir, "ref.md", "---\n---\n\nSee [[Nickname]].\n")

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	var ref *EvalContext
	for _, n := range notes {
		if n.File.Name == "ref.md" {
			ref = n
		}
	}
	if got := mustEval(t, `file.hasLink("Nickname")`, ref); got != true {
		t.Errorf(`file.hasLink("Nickname") = %v`, got)
	}
	if got := mustEval(t, `file("Nickname").name`, ref); got != "Full Name.md" {
		t.Errorf(`file("Nickname").name = %v, want Full Name.md (alias resolved)`, got)
	}
}

// TestLoadVault_AssetResolves confirms a wikilink/embed targeting a
// non-.md file (an image, here) resolves to something, even though
// ParseDir never walks it as a note — the real, extremely common case
// of a ![[cover.webp]] embed.
func TestLoadVault_AssetResolves(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "a.md", "---\n---\n\n![[cover.webp]]\n")
	if err := os.WriteFile(filepath.Join(dir, "cover.webp"), []byte("not a real image"), 0644); err != nil {
		t.Fatal(err)
	}

	notes, errs := LoadVault(dir)
	if len(errs) > 0 {
		t.Fatalf("LoadVault: %v", errs)
	}
	a := notes[0]
	if len(a.File.Embeds) != 1 || a.File.Embeds[0] != "cover.webp" {
		t.Fatalf("a.File.Embeds = %v, want [cover.webp]", a.File.Embeds)
	}
	resolved := mustEval(t, `file("cover.webp")`, a)
	fr, ok := resolved.(fileRef)
	if !ok || fr.File.Name != "cover.webp" || fr.File.Ext != "webp" {
		t.Fatalf(`file("cover.webp") = %#v`, resolved)
	}
}

func TestLoadVault_ToleratesBadNote(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "good.md", "---\nstatus: open\n---\n\nbody\n")
	writeNote(t, dir, "template.md", "---\ncreated: {{date}}\n---\n\nbody\n")

	notes, errs := LoadVault(dir)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if len(notes) != 1 || notes[0].File.Name != "good.md" {
		t.Fatalf("notes = %+v, want just good.md", notes)
	}
}

// TestQuery_ResilientToPerNoteFilterError reproduces a real crash found
// running kepano-obsidian's actual Books.base: its filter calls
// categories.contains(...), and most notes simply have no "categories"
// property at all (nil, not a list). Before the nil.contains() => false
// fix, that one call errored and the error aborted the ENTIRE query —
// every other note's results lost because of one unrelated note.
func TestQuery_ResilientToPerNoteFilterError(t *testing.T) {
	dir := t.TempDir()
	writeNote(t, dir, "has-categories.md", "---\ncategories: [a, b]\n---\n\nbody\n")
	writeNote(t, dir, "no-categories.md", "---\nother: 1\n---\n\nbody\n")

	base := &Base{Filters: &Filter{Statement: `categories.contains("a")`}}
	results, warnings, err := Query(base, dir, "")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("Query warnings: %v (expected none now that nil.contains() => false)", warnings)
	}
	if len(results) != 1 || results[0].File.Name != "has-categories.md" {
		t.Errorf("Query results = %+v, want just has-categories.md", results)
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

	results, warnings, err := Query(base, dir, "books")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(warnings) > 0 {
		t.Fatalf("Query warnings: %v", warnings)
	}
	if len(results) != 1 || results[0].File.Name != "a.md" {
		t.Errorf("Query results = %+v, want just a.md", results)
	}

	all, _, err := Query(base, dir, "")
	if err != nil {
		t.Fatalf("Query (default view): %v", err)
	}
	if len(all) != 1 {
		t.Errorf("Query with default (first) view = %d results, want 1", len(all))
	}

	if _, _, err := Query(base, dir, "nonexistent"); err == nil {
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

	res, warnings, err := Lint(base, dir)
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if len(warnings) > 0 {
		t.Fatalf("Lint warnings: %v", warnings)
	}
	if res.PopulationCount != 2 || res.SatisfiedCount != 2 || !res.Passed {
		t.Errorf("res = %+v", res)
	}

	plain, err := Parse([]byte("views:\n  - type: table\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, _, err := Lint(plain, dir); err == nil {
		t.Error("expected an error linting a base with no sillAssert extension")
	}
}

func booksFixture() []*EvalContext {
	return []*EvalContext{
		{
			File: FileMeta{Path: "Snow Crash.md"},
			Note: map[string]any{
				"status": "read",
				"author": "Neal Stephenson",
				"year":   1992.0,
				"genre":  []any{"sci-fi", "cyberpunk"},
			},
		},
		{
			File: FileMeta{Path: "Sapiens.md"},
			Note: map[string]any{
				"status": "reading",
				"author": "Yuval Noah Harari",
				"year":   2011.0,
				"genre":  []any{"nonfiction", "history"},
			},
		},
	}
}

func TestFormatQueryResults_Table_NestsViaIndentation(t *testing.T) {
	out, err := FormatQueryResults(booksFixture(), RenderConfig{Format: "table"})
	if err != nil {
		t.Fatalf("FormatQueryResults: %v", err)
	}
	want := "Snow Crash.md\n" +
		"  author: Neal Stephenson\n" +
		"  genre:\n" +
		"    - sci-fi\n" +
		"    - cyberpunk\n" +
		"  status: read\n" +
		"  year:   1992\n" +
		"\n" +
		"Sapiens.md\n" +
		"  author: Yuval Noah Harari\n" +
		"  genre:\n" +
		"    - nonfiction\n" +
		"    - history\n" +
		"  status: reading\n" +
		"  year:   2011\n"
	if out != want {
		t.Errorf("table output =\n%q\nwant\n%q", out, want)
	}
}

func TestFormatQueryResults_Grid_FlattensNestingInline(t *testing.T) {
	out, err := FormatQueryResults(booksFixture(), RenderConfig{
		Format:  "grid",
		Columns: []string{"status", "author", "genre"},
	})
	if err != nil {
		t.Fatalf("FormatQueryResults: %v", err)
	}
	if !strings.Contains(out, "[sci-fi, cyberpunk]") {
		t.Errorf("expected genre flattened inline, got:\n%s", out)
	}
	if strings.Contains(out, "\n    - ") {
		t.Errorf("grid must not indent nested values like table does, got:\n%s", out)
	}
}

// TestFormatQueryResults_Grid_ResolvesFileColumns reproduces a real bug
// found running kepano-obsidian's actual Evergreen.base: its view's
// order: names file.name and created — file.name is a file.* reference,
// not a bare frontmatter property, so a raw map lookup on Note silently
// returned nothing for it. Columns now go through the real evaluator.
func TestFormatQueryResults_Grid_ResolvesFileColumns(t *testing.T) {
	notes := []*EvalContext{{
		File: FileMeta{Path: "sub/Evergreen notes.md", Name: "Evergreen notes.md"},
		Note: map[string]any{"created": "2023-09-14"},
	}}
	out, err := FormatQueryResults(notes, RenderConfig{
		Format:  "grid",
		Columns: []string{"file.name", "created"},
	})
	if err != nil {
		t.Fatalf("FormatQueryResults: %v", err)
	}
	if !strings.Contains(out, "Evergreen notes.md") {
		t.Errorf("expected file.name to resolve to the note's filename, got:\n%s", out)
	}
}

func TestFormatQueryResults_JSON_NestsNatively(t *testing.T) {
	out, err := FormatQueryResults(booksFixture(), RenderConfig{Format: "json"})
	if err != nil {
		t.Fatalf("FormatQueryResults: %v", err)
	}
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output isn't valid JSON: %v\n%s", err, out)
	}
	genre, ok := parsed[0]["genre"].([]any)
	if !ok || len(genre) != 2 || genre[0] != "sci-fi" {
		t.Errorf("genre = %#v, want a real [sci-fi cyberpunk] array", parsed[0]["genre"])
	}
}

func TestFormatQueryResults_TOON_FlattensAndEscapesCommas(t *testing.T) {
	out, err := FormatQueryResults(booksFixture(), RenderConfig{
		Format:  "toon",
		Columns: []string{"status", "genre"},
	})
	if err != nil {
		t.Fatalf("FormatQueryResults: %v", err)
	}
	if !strings.HasPrefix(out, "notes[2]{path,status,genre}:\n") {
		t.Errorf("unexpected TOON header, got:\n%s", out)
	}
	// genre's own flattened "," forces CSV-style quoting of the whole cell.
	if !strings.Contains(out, `"[sci-fi, cyberpunk]"`) {
		t.Errorf("expected the comma-containing cell to be quoted, got:\n%s", out)
	}
}

func TestFormatQueryResults_ColumnsDefaultToEveryKeySorted(t *testing.T) {
	out, err := FormatQueryResults(booksFixture(), RenderConfig{Format: "grid"})
	if err != nil {
		t.Fatalf("FormatQueryResults: %v", err)
	}
	header := strings.SplitN(out, "\n", 2)[0]
	if header != "path           author             genre                  status   year" {
		t.Errorf("header = %q", header)
	}
}

func TestFormatQueryResults_UnknownFormat(t *testing.T) {
	if _, err := FormatQueryResults(booksFixture(), RenderConfig{Format: "bogus"}); err == nil {
		t.Error("expected an error for an unknown format")
	}
}
