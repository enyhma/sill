package sill

import (
	"testing"
	"time"
)

func mustEval(t *testing.T, expr string, ctx *EvalContext) any {
	t.Helper()
	v, err := EvalExpr(expr, ctx)
	if err != nil {
		t.Fatalf("EvalExpr(%q): %v", expr, err)
	}
	return v
}

func TestEval_ArithmeticAndComparison(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"price": 19.99, "age": 2.0}}

	cases := map[string]any{
		`price`:                  19.99,
		`price + 1`:              20.99,
		`price - 1`:              18.99,
		`age * 10`:               20.0,
		`age / 2`:                1.0,
		`5 % 2`:                  1.0,
		`price > 10`:             true,
		`price <= 19.99`:         true,
		`price == 19.99`:         true,
		`price != 19.99`:         false,
		`!(price > 100)`:         true,
		`price > 10 && age > 1`:  true,
		`price > 100 || age > 1`: true,
	}
	for expr, want := range cases {
		got := mustEval(t, expr, ctx)
		if got != want {
			t.Errorf("%s = %v (%T), want %v (%T)", expr, got, got, want, want)
		}
	}
}

func TestEval_StringConcatenationAndFormula(t *testing.T) {
	ctx := &EvalContext{
		Note: map[string]any{"price": 19.99},
		Formulas: map[string]string{
			"formatted_price": `if(price, price.toFixed(2) + " dollars")`,
			"ppu":             `(price / 4).toFixed(2)`,
		},
	}

	if got := mustEval(t, `formula.formatted_price`, ctx); got != "19.99 dollars" {
		t.Errorf("formula.formatted_price = %q", got)
	}
	if got := mustEval(t, `formula.ppu`, ctx); got != "5.00" {
		t.Errorf("formula.ppu = %q", got)
	}
}

func TestEval_UndefinedFormulaFailsLoudly(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{}}
	_, err := EvalExpr("formula.nonexistent", ctx)
	if err == nil {
		t.Fatal("expected an error for an undefined formula, got nil")
	}
}

func TestEval_PropertyResolutionForms(t *testing.T) {
	ctx := &EvalContext{
		Note: map[string]any{
			"status": "done",
			"links": map[string]any{
				"refines": []any{"CTX-1"},
			},
		},
		File: FileMeta{Name: "note.md", Path: "backlog/note.md"},
	}

	if got := mustEval(t, `status`, ctx); got != "done" {
		t.Errorf("status = %v", got)
	}
	if got := mustEval(t, `note.status`, ctx); got != "done" {
		t.Errorf("note.status = %v", got)
	}
	if got := mustEval(t, `note["status"]`, ctx); got != "done" {
		t.Errorf(`note["status"] = %v`, got)
	}
	if got := mustEval(t, `file.name`, ctx); got != "note.md" {
		t.Errorf("file.name = %v", got)
	}
	refines := mustEval(t, `links.refines`, ctx)
	list, ok := refines.([]any)
	if !ok || len(list) != 1 || list[0] != "CTX-1" {
		t.Errorf("links.refines = %#v", refines)
	}
}

func TestEval_DateArithmetic(t *testing.T) {
	base := time.Date(2026, 1, 15, 0, 0, 0, 0, time.Local)

	plusMonth, err := EvalExpr(`d + "1M"`, &EvalContext{Note: map[string]any{"d": base}})
	if err != nil {
		t.Fatalf("d + 1M: %v", err)
	}
	got := plusMonth.(time.Time)
	want := base.AddDate(0, 1, 0)
	if !got.Equal(want) {
		t.Errorf("d + \"1M\" = %v, want %v", got, want)
	}

	minusHours, err := EvalExpr(`d - "2h"`, &EvalContext{Note: map[string]any{"d": base}})
	if err != nil {
		t.Fatalf("d - 2h: %v", err)
	}
	if !minusHours.(time.Time).Equal(base.Add(-2 * time.Hour)) {
		t.Errorf("d - \"2h\" = %v", minusHours)
	}

	later := base.AddDate(0, 0, 10)
	diff, err := EvalExpr(`a - b`, &EvalContext{Note: map[string]any{"a": later, "b": base}})
	if err != nil {
		t.Fatalf("a - b: %v", err)
	}
	d, ok := diff.(duration)
	if !ok || d.Clock != 10*24*time.Hour {
		t.Errorf("a - b = %#v, want a 10-day duration", diff)
	}
}

func TestEval_DurationUnitSuffixes(t *testing.T) {
	cases := map[string]duration{
		"1y":       {Years: 1},
		"2 years":  {Years: 2},
		"1M":       {Months: 1},
		"3 months": {Months: 3},
		"1w":       {Days: 7},
		"2 weeks":  {Days: 14},
		"5d":       {Days: 5},
		"5 days":   {Days: 5},
		"3h":       {Clock: 3 * time.Hour},
		"1m":       {Clock: time.Minute},
		"30s":      {Clock: 30 * time.Second},
	}
	for s, want := range cases {
		got, err := parseDuration(s)
		if err != nil {
			t.Fatalf("parseDuration(%q): %v", s, err)
		}
		if got != want {
			t.Errorf("parseDuration(%q) = %+v, want %+v", s, got, want)
		}
	}
}

func TestEval_DateFormat(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"d": time.Date(2026, 3, 5, 0, 0, 0, 0, time.Local)}}
	got := mustEval(t, `d.format("YYYY-MM-DD")`, ctx)
	if got != "2026-03-05" {
		t.Errorf("format = %v", got)
	}
}

func TestEvalFilter_AndOrNot(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"status": "done", "age": 5.0}}

	f := &Filter{And: []Filter{
		{Statement: `status == "done"`},
		{Not: []Filter{{Statement: `age > 10`}}},
	}}
	ok, err := EvalFilter(f, ctx)
	if err != nil {
		t.Fatalf("EvalFilter: %v", err)
	}
	if !ok {
		t.Error("expected filter to pass")
	}

	f2 := &Filter{Or: []Filter{
		{Statement: `status == "archived"`},
		{Statement: `age > 10`},
	}}
	ok2, err := EvalFilter(f2, ctx)
	if err != nil {
		t.Fatalf("EvalFilter: %v", err)
	}
	if ok2 {
		t.Error("expected filter to fail")
	}
}
