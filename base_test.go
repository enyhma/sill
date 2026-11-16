package sill

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

// docExample is the full example assembled from Obsidian's own Bases
// syntax documentation (https://obsidian.md/help/bases/syntax): the
// filters, formulas, properties, summaries, and views sections as shown
// there, combined into one file.
const docExample = `
filters:
  or:
    - file.hasTag("tag")
    - and:
        - file.hasTag("book")
        - file.hasLink("Textbook")
    - not:
        - file.hasTag("book")
formulas:
  formatted_price: 'if(price, price.toFixed(2) + " dollars")'
  ppu: "(price / age).toFixed(2)"
properties:
  status:
    displayName: Status
  formula.formatted_price:
    displayName: "Price"
  file.ext:
    displayName: Extension
summaries:
  customAverage: 'values.mean().round(3)'
views:
  - type: table
    name: "My table"
    limit: 10
    groupBy:
      property: note.age
      direction: DESC
    filters:
      and:
        - 'status != "done"'
    order:
      - file.name
      - note.age
    summaries:
      formula.ppu: Average
`

func TestParse_DocExample(t *testing.T) {
	b, err := Parse([]byte(docExample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if b.Filters == nil || len(b.Filters.Or) != 3 {
		t.Fatalf("Filters.Or: got %+v, want 3 entries", b.Filters)
	}
	if got := b.Filters.Or[0].Statement; got != `file.hasTag("tag")` {
		t.Errorf("Filters.Or[0].Statement = %q", got)
	}
	if got := b.Filters.Or[1].And; len(got) != 2 {
		t.Errorf("Filters.Or[1].And = %+v, want 2 entries", got)
	}
	if got := b.Filters.Or[2].Not; len(got) != 1 {
		t.Errorf("Filters.Or[2].Not = %+v, want 1 entry", got)
	}

	if got := b.Formulas["ppu"]; got != `(price / age).toFixed(2)` {
		t.Errorf("Formulas[ppu] = %q", got)
	}

	if got := b.Properties["status"]["displayName"]; got != "Status" {
		t.Errorf("Properties[status][displayName] = %v", got)
	}

	if got := b.Summaries["customAverage"]; got != "values.mean().round(3)" {
		t.Errorf("Summaries[customAverage] = %q", got)
	}

	if len(b.Views) != 1 {
		t.Fatalf("Views: got %d, want 1", len(b.Views))
	}
	v := b.Views[0]
	if v.Type != "table" || v.Name != "My table" || v.Limit != 10 {
		t.Errorf("view basic fields: %+v", v)
	}
	if v.GroupBy == nil || v.GroupBy.Property != "note.age" || v.GroupBy.Direction != "DESC" {
		t.Errorf("view.GroupBy: %+v", v.GroupBy)
	}
	if v.Filters == nil || len(v.Filters.And) != 1 || v.Filters.And[0].Statement != `status != "done"` {
		t.Errorf("view.Filters: %+v", v.Filters)
	}
	if !reflect.DeepEqual(v.Order, []string{"file.name", "note.age"}) {
		t.Errorf("view.Order: %+v", v.Order)
	}
	if v.Summaries["formula.ppu"] != "Average" {
		t.Errorf("view.Summaries: %+v", v.Summaries)
	}
}

func TestParse_RoundTrips(t *testing.T) {
	b1, err := Parse([]byte(docExample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out, err := yaml.Marshal(b1)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	b2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-Parse after Marshal: %v\n%s", err, out)
	}

	if !reflect.DeepEqual(b1, b2) {
		t.Errorf("round-trip mismatch:\nbefore: %+v\nafter:  %+v\nmarshaled:\n%s", b1, b2, out)
	}
}

func TestParse_UnknownTopLevelKeyPreserved(t *testing.T) {
	content := `
sillAssert:
  population: 'status == "done"'
  condition: 'file.hasProperty("reviewed")'
  threshold: 0.9
views:
  - type: table
`
	b, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := b.Extra["sillAssert"]; !ok {
		t.Fatalf("expected sillAssert preserved in Extra, got keys %v", mapKeys(b.Extra))
	}

	out, err := yaml.Marshal(b)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	b2, err := Parse(out)
	if err != nil {
		t.Fatalf("re-Parse: %v\n%s", err, out)
	}
	var threshold float64
	sillAssert := b2.Extra["sillAssert"]
	if err := sillAssert.Decode(&struct {
		Threshold *float64 `yaml:"threshold"`
	}{Threshold: &threshold}); err != nil {
		t.Fatalf("decoding round-tripped sillAssert: %v", err)
	}
	if threshold != 0.9 {
		t.Errorf("threshold after round-trip = %v, want 0.9", threshold)
	}
}

func TestParse_MalformedYAML(t *testing.T) {
	if _, err := Parse([]byte("filters: [unterminated")); err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestParse_UnknownFilterKey(t *testing.T) {
	_, err := Parse([]byte("filters:\n  xor:\n    - a\n    - b\n"))
	if err == nil {
		t.Fatal("expected error for unknown filter key, got nil")
	}
}

func mapKeys(m map[string]yaml.Node) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
