package sill

import "fmt"

// sillAssertKey is the sidecar top-level key sill's assertion extension
// lives under — never filters:/formulas:/views:, so a file using it stays
// a valid, correctly-rendering Base if opened in Obsidian (see
// landmarks/design/engineering/DEC-eNLURTFi6w).
const sillAssertKey = "sillAssert"

// Assertion is sill's one real addition to the Bases spec: given a
// Population (the notes to consider) and a Condition (what must hold),
// compute the count or percentage of Population-matching notes that also
// satisfy Condition, and compare against Threshold.
type Assertion struct {
	Population Filter  `yaml:"population"`
	Condition  Filter  `yaml:"condition"`
	Threshold  float64 `yaml:"threshold"`
	Mode       string  `yaml:"mode"` // "count" or "percentage" (default)
}

// ParseAssertion decodes a Base's sillAssert sidecar, if present. Returns
// (nil, nil) when the key is absent — a plain Base with no assertion.
func ParseAssertion(b *Base) (*Assertion, error) {
	node, ok := b.Extra[sillAssertKey]
	if !ok {
		return nil, nil
	}
	var a Assertion
	if err := node.Decode(&a); err != nil {
		return nil, fmt.Errorf("%s: %w", sillAssertKey, err)
	}
	if a.Mode == "" {
		a.Mode = "percentage"
	}
	if a.Mode != "count" && a.Mode != "percentage" {
		return nil, fmt.Errorf("%s: unknown mode %q (expected count or percentage)", sillAssertKey, a.Mode)
	}
	return &a, nil
}

// AssertionResult is the outcome of evaluating an Assertion against a set
// of notes.
type AssertionResult struct {
	PopulationCount int
	SatisfiedCount  int
	Percentage      float64 // SatisfiedCount / PopulationCount; 0 if PopulationCount is 0
	Passed          bool
}

// EvalAssertion evaluates a's population filter and condition against
// every note in ctxs.
func EvalAssertion(a *Assertion, ctxs []*EvalContext) (*AssertionResult, error) {
	res := &AssertionResult{}
	for _, ctx := range ctxs {
		inPop, err := EvalFilter(&a.Population, ctx)
		if err != nil {
			return nil, err
		}
		if !inPop {
			continue
		}
		res.PopulationCount++
		ok, err := EvalFilter(&a.Condition, ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			res.SatisfiedCount++
		}
	}
	if res.PopulationCount > 0 {
		res.Percentage = float64(res.SatisfiedCount) / float64(res.PopulationCount)
	}
	if a.Mode == "count" {
		res.Passed = float64(res.SatisfiedCount) >= a.Threshold
	} else {
		res.Passed = res.Percentage >= a.Threshold
	}
	return res, nil
}

// ExitCode maps an AssertionResult to sill's documented, stable lint exit
// code: 0 when passed, 1 otherwise. Unlike Obsidian's own CLI, whose docs
// don't specify exit-code behavior, this is part of sill's contract.
func (r *AssertionResult) ExitCode() int {
	if r.Passed {
		return 0
	}
	return 1
}
