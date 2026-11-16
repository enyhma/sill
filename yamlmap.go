package sill

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// decodeMapping walks a YAML mapping node, calling known for each key/value
// pair. known returns handled=true if it consumed the pair; unhandled pairs
// are returned so the caller can preserve them (e.g. as an Extra field).
func decodeMapping(node *yaml.Node, known func(key string, val *yaml.Node) (handled bool, err error)) (map[string]yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected a mapping, got %v", node.Kind)
	}
	var extra map[string]yaml.Node
	for i := 0; i < len(node.Content); i += 2 {
		key, val := node.Content[i], node.Content[i+1]
		handled, err := known(key.Value, val)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key.Value, err)
		}
		if !handled {
			if extra == nil {
				extra = make(map[string]yaml.Node)
			}
			extra[key.Value] = *val
		}
	}
	return extra, nil
}

// encodeMapping builds a mapping node from ordered known key/value pairs
// plus any preserved extra pairs.
func encodeMapping(known []keyValue, extra map[string]yaml.Node) (*yaml.Node, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, kv := range known {
		var k, v yaml.Node
		if err := k.Encode(kv.key); err != nil {
			return nil, err
		}
		if err := v.Encode(kv.value); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, &k, &v)
	}
	for key, val := range extra {
		var k yaml.Node
		if err := k.Encode(key); err != nil {
			return nil, err
		}
		v := val
		node.Content = append(node.Content, &k, &v)
	}
	return node, nil
}

type keyValue struct {
	key   string
	value any
}
