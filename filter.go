package sill

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Filter is a node in a filters: tree: either a single expression
// statement, or an and/or/not of nested Filters.
type Filter struct {
	Statement string
	And       []Filter
	Or        []Filter
	Not       []Filter
}

func (f *Filter) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Decode(&f.Statement)
	case yaml.MappingNode:
		if len(node.Content) != 2 {
			return fmt.Errorf("filter mapping must have exactly one of and/or/not, got %d keys", len(node.Content)/2)
		}
		key, val := node.Content[0].Value, node.Content[1]
		switch key {
		case "and":
			return val.Decode(&f.And)
		case "or":
			return val.Decode(&f.Or)
		case "not":
			return val.Decode(&f.Not)
		default:
			return fmt.Errorf("unknown filter key %q (expected and/or/not)", key)
		}
	default:
		return fmt.Errorf("filter must be a string statement or an and/or/not mapping, got %v", node.Kind)
	}
}

func (f Filter) MarshalYAML() (any, error) {
	switch {
	case f.And != nil:
		return map[string]any{"and": f.And}, nil
	case f.Or != nil:
		return map[string]any{"or": f.Or}, nil
	case f.Not != nil:
		return map[string]any{"not": f.Not}, nil
	default:
		return f.Statement, nil
	}
}
