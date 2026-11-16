package sill

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Base is a parsed .base file: the five documented top-level keys, plus
// any unrecognized ones preserved verbatim so a round-trip never loses
// data (e.g. a future Obsidian addition, or sill's own sidecar key — see
// landmarks/design/engineering/CAP-z3ISN9O6c3).
type Base struct {
	Filters    *Filter
	Formulas   map[string]string
	Properties map[string]map[string]any
	Summaries  map[string]string
	Views      []View
	Extra      map[string]yaml.Node
}

func (b *Base) UnmarshalYAML(node *yaml.Node) error {
	extra, err := decodeMapping(node, func(key string, val *yaml.Node) (bool, error) {
		switch key {
		case "filters":
			return true, val.Decode(&b.Filters)
		case "formulas":
			return true, val.Decode(&b.Formulas)
		case "properties":
			return true, val.Decode(&b.Properties)
		case "summaries":
			return true, val.Decode(&b.Summaries)
		case "views":
			return true, val.Decode(&b.Views)
		default:
			return false, nil
		}
	})
	if err != nil {
		return err
	}
	b.Extra = extra
	return nil
}

func (b Base) MarshalYAML() (any, error) {
	var known []keyValue
	add := func(key string, value any, omit bool) {
		if !omit {
			known = append(known, keyValue{key, value})
		}
	}
	add("filters", b.Filters, b.Filters == nil)
	add("formulas", b.Formulas, len(b.Formulas) == 0)
	add("properties", b.Properties, len(b.Properties) == 0)
	add("summaries", b.Summaries, len(b.Summaries) == 0)
	add("views", b.Views, len(b.Views) == 0)
	return encodeMapping(known, b.Extra)
}

// Parse decodes raw .base YAML content into a Base.
func Parse(content []byte) (*Base, error) {
	var b Base
	if err := yaml.Unmarshal(content, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// ParseFile reads and parses the .base file at path.
func ParseFile(path string) (*Base, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(content)
}
