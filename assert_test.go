package sill

import "testing"

func notesWithStatus(statuses ...string) []*EvalContext {
	out := make([]*EvalContext, len(statuses))
	for i, s := range statuses {
		out[i] = &EvalContext{Note: map[string]any{"status": s}}
	}
	return out
}

func TestEvalAssertion_Percentage(t *testing.T) {
	a := &Assertion{
		Population: Filter{Statement: `status != "draft"`},
		Condition:  Filter{Statement: `status == "reviewed"`},
		Mode:       "percentage",
		Threshold:  0.75,
	}
	// population: 4 non-draft notes; condition: 3 of them reviewed -> 0.75
	notes := notesWithStatus("reviewed", "reviewed", "reviewed", "open", "draft")

	res, err := EvalAssertion(a, notes)
	if err != nil {
		t.Fatalf("EvalAssertion: %v", err)
	}
	if res.PopulationCount != 4 {
		t.Errorf("PopulationCount = %d, want 4", res.PopulationCount)
	}
	if res.SatisfiedCount != 3 {
		t.Errorf("SatisfiedCount = %d, want 3", res.SatisfiedCount)
	}
	if res.Percentage != 0.75 {
		t.Errorf("Percentage = %v, want 0.75", res.Percentage)
	}
	if !res.Passed {
		t.Error("expected threshold 0.75 to pass at exactly 0.75")
	}
	if res.ExitCode() != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode())
	}

	a.Threshold = 0.9
	res2, err := EvalAssertion(a, notes)
	if err != nil {
		t.Fatalf("EvalAssertion: %v", err)
	}
	if res2.Passed {
		t.Error("expected threshold 0.9 to fail at 0.75")
	}
	if res2.ExitCode() != 1 {
		t.Errorf("ExitCode = %d, want 1", res2.ExitCode())
	}
}

func TestEvalAssertion_Count(t *testing.T) {
	a := &Assertion{
		Population: Filter{Statement: `status != "draft"`},
		Condition:  Filter{Statement: `status == "reviewed"`},
		Mode:       "count",
		Threshold:  3,
	}
	notes := notesWithStatus("reviewed", "reviewed", "reviewed", "open")

	res, err := EvalAssertion(a, notes)
	if err != nil {
		t.Fatalf("EvalAssertion: %v", err)
	}
	if res.SatisfiedCount != 3 || !res.Passed {
		t.Errorf("res = %+v, want SatisfiedCount=3, Passed=true", res)
	}
}

func TestEvalAssertion_EmptyPopulation(t *testing.T) {
	a := &Assertion{
		Population: Filter{Statement: `status == "nonexistent"`},
		Condition:  Filter{Statement: `status == "reviewed"`},
		Threshold:  0,
	}
	notes := notesWithStatus("open")

	res, err := EvalAssertion(a, notes)
	if err != nil {
		t.Fatalf("EvalAssertion: %v", err)
	}
	if res.PopulationCount != 0 || res.Percentage != 0 {
		t.Errorf("res = %+v, want an empty population", res)
	}
	if !res.Passed {
		t.Error("expected a zero threshold to pass against an empty population")
	}
}

func TestParseAssertion_SidecarRoundTrip(t *testing.T) {
	content := `
views:
  - type: table
sillAssert:
  population: 'status != "draft"'
  condition: 'status == "reviewed"'
  threshold: 0.75
`
	b, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// The file still parses as a plain Base — the sidecar is just extra
	// data, not a required/recognized top-level key.
	if len(b.Views) != 1 {
		t.Fatalf("expected the base to still parse its views, got %+v", b.Views)
	}

	a, err := ParseAssertion(b)
	if err != nil {
		t.Fatalf("ParseAssertion: %v", err)
	}
	if a == nil {
		t.Fatal("expected a non-nil Assertion")
	}
	if a.Mode != "percentage" {
		t.Errorf("Mode = %q, want default percentage", a.Mode)
	}
	if a.Threshold != 0.75 {
		t.Errorf("Threshold = %v, want 0.75", a.Threshold)
	}
	if a.Population.Statement != `status != "draft"` {
		t.Errorf("Population = %+v", a.Population)
	}

	notes := notesWithStatus("reviewed", "reviewed", "reviewed", "open")
	res, err := EvalAssertion(a, notes)
	if err != nil {
		t.Fatalf("EvalAssertion: %v", err)
	}
	if !res.Passed {
		t.Errorf("res = %+v, want passed", res)
	}
}

func TestParseAssertion_AbsentSidecar(t *testing.T) {
	b, err := Parse([]byte("views:\n  - type: table\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	a, err := ParseAssertion(b)
	if err != nil {
		t.Fatalf("ParseAssertion: %v", err)
	}
	if a != nil {
		t.Errorf("expected nil Assertion for a plain Base, got %+v", a)
	}
}

func TestParseAssertion_UnknownMode(t *testing.T) {
	content := `
sillAssert:
  population: 'status == "x"'
  condition: 'status == "y"'
  threshold: 1
  mode: bogus
`
	b, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := ParseAssertion(b); err == nil {
		t.Fatal("expected an error for an unknown mode, got nil")
	}
}
