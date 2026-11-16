package sill

import "gopkg.in/yaml.v3"

// GroupBy configures a view's grouping: the property to group by and the
// sort direction of the groups (ASC/DESC).
type GroupBy struct {
	Property  string `yaml:"property"`
	Direction string `yaml:"direction"`
}

// View is one entry in a .base file's views: list. Beyond the
// filter/sort/group fields sill evaluates, a view may carry rendering-only
// keys specific to its type (e.g. Cards' image/imageFit) — those are
// preserved in Extra rather than interpreted, per sill's non-goal of view
// rendering.
type View struct {
	Type       string
	Name       string
	Limit      int
	GroupBy    *GroupBy
	GroupOrder []*string
	Filters    *Filter
	Order      []string
	Summaries  map[string]string
	Extra      map[string]yaml.Node
}

func (v *View) UnmarshalYAML(node *yaml.Node) error {
	extra, err := decodeMapping(node, func(key string, val *yaml.Node) (bool, error) {
		switch key {
		case "type":
			return true, val.Decode(&v.Type)
		case "name":
			return true, val.Decode(&v.Name)
		case "limit":
			return true, val.Decode(&v.Limit)
		case "groupBy":
			return true, val.Decode(&v.GroupBy)
		case "groupOrder":
			return true, val.Decode(&v.GroupOrder)
		case "filters":
			return true, val.Decode(&v.Filters)
		case "order":
			return true, val.Decode(&v.Order)
		case "summaries":
			return true, val.Decode(&v.Summaries)
		default:
			return false, nil
		}
	})
	if err != nil {
		return err
	}
	v.Extra = extra
	return nil
}

func (v View) MarshalYAML() (any, error) {
	var known []keyValue
	add := func(key string, value any, omit bool) {
		if !omit {
			known = append(known, keyValue{key, value})
		}
	}
	add("type", v.Type, v.Type == "")
	add("name", v.Name, v.Name == "")
	add("limit", v.Limit, v.Limit == 0)
	add("groupBy", v.GroupBy, v.GroupBy == nil)
	add("groupOrder", v.GroupOrder, len(v.GroupOrder) == 0)
	add("filters", v.Filters, v.Filters == nil)
	add("order", v.Order, len(v.Order) == 0)
	add("summaries", v.Summaries, len(v.Summaries) == 0)
	return encodeMapping(known, v.Extra)
}
