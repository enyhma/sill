package sill

import (
	"testing"
	"time"
)

func evalWith(t *testing.T, expr string, note map[string]any, file FileMeta) any {
	t.Helper()
	return mustEval(t, expr, &EvalContext{Note: note, File: file})
}

func TestFunctions_Any(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"x": "hello"}}
	if got := mustEval(t, `x.isType("string")`, ctx); got != true {
		t.Errorf("isType = %v", got)
	}
	if got := mustEval(t, `x.toString()`, ctx); got != "hello" {
		t.Errorf("toString = %v", got)
	}
	if got := mustEval(t, `(0).isTruthy()`, ctx); got != false {
		t.Errorf("isTruthy(0) = %v", got)
	}
}

func TestFunctions_String(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"s": "Hello World"}}
	cases := map[string]any{
		`s.contains("World")`:             true,
		`s.containsAll("Hello", "World")`: true,
		`s.containsAny("xyz", "World")`:   true,
		`s.endsWith("World")`:             true,
		`s.startsWith("Hello")`:           true,
		`s.lower()`:                       "hello world",
		`s.trim()`:                        "Hello World",
		`s.reverse()`:                     "dlroW olleH",
		`s.replace("World", "There")`:     "Hello There",
		`s.slice(0, 5)`:                   "Hello",
		`s.length`:                        11.0,
		`"".isEmpty()`:                    true,
		`s.title()`:                       "Hello World",
	}
	for expr, want := range cases {
		got := mustEval(t, expr, ctx)
		if got != want {
			t.Errorf("%s = %v (%T), want %v (%T)", expr, got, got, want, want)
		}
	}
}

func TestFunctions_Number(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{}}
	cases := map[string]any{
		`(-5).abs()`:       5.0,
		`(2.3).ceil()`:     3.0,
		`(2.7).floor()`:    2.0,
		`(2.345).round(2)`: 2.35,
		`(2.5).round(0)`:   3.0,
	}
	for expr, want := range cases {
		got := mustEval(t, expr, ctx)
		if got != want {
			t.Errorf("%s = %v, want %v", expr, got, want)
		}
	}
	if got, err := EvalExpr(`number("42")`, ctx); err != nil || got != 42.0 {
		t.Errorf("number(\"42\") = %v, %v", got, err)
	}
}

func TestFunctions_List(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"l": []any{"a", "b", "c"}}}
	if got := mustEval(t, `l.contains("b")`, ctx); got != true {
		t.Errorf("contains = %v", got)
	}
	if got := mustEval(t, `l.containsAll("a", "c")`, ctx); got != true {
		t.Errorf("containsAll = %v", got)
	}
	if got := mustEval(t, `l.join(", ")`, ctx); got != "a, b, c" {
		t.Errorf("join = %v", got)
	}
	if got := mustEval(t, `l.length`, ctx); got != 3.0 {
		t.Errorf("length = %v", got)
	}
	rev := mustEval(t, `l.reverse()`, ctx).([]any)
	if rev[0] != "c" || rev[2] != "a" {
		t.Errorf("reverse = %v", rev)
	}
	sliced := mustEval(t, `l.slice(1, 3)`, ctx).([]any)
	if len(sliced) != 2 || sliced[0] != "b" || sliced[1] != "c" {
		t.Errorf("slice = %v", sliced)
	}

	ctx2 := &EvalContext{Note: map[string]any{"l": []any{3.0, 1.0, 2.0}}}
	sorted := mustEval(t, `l.sort()`, ctx2).([]any)
	if sorted[0] != 1.0 || sorted[1] != 2.0 || sorted[2] != 3.0 {
		t.Errorf("sort = %v", sorted)
	}

	ctx3 := &EvalContext{Note: map[string]any{"l": []any{1.0, 1.0, 2.0}}}
	uniq := mustEval(t, `l.unique()`, ctx3).([]any)
	if len(uniq) != 2 {
		t.Errorf("unique = %v", uniq)
	}

	ctx4 := &EvalContext{Note: map[string]any{"l": []any{}}}
	if got := mustEval(t, `l.isEmpty()`, ctx4); got != true {
		t.Errorf("isEmpty = %v", got)
	}
}

func TestFunctions_Object(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"o": map[string]any{"a": 1.0, "b": 2.0}}}
	if got := mustEval(t, `o.isEmpty()`, ctx); got != false {
		t.Errorf("isEmpty = %v", got)
	}
	keys := mustEval(t, `o.keys()`, ctx).([]any)
	if len(keys) != 2 {
		t.Errorf("keys = %v", keys)
	}
}

func TestFunctions_File(t *testing.T) {
	file := FileMeta{
		Name: "note.md", Path: "backlog/note.md", Folder: "backlog",
		Tags: []string{"todo", "urgent"}, Links: []string{"Other Note"},
	}
	note := map[string]any{"status": "open"}

	if got := evalWith(t, `file.hasTag("urgent")`, note, file); got != true {
		t.Errorf("hasTag = %v", got)
	}
	if got := evalWith(t, `file.hasTag("missing")`, note, file); got != false {
		t.Errorf("hasTag(missing) = %v", got)
	}
	if got := evalWith(t, `file.inFolder("backlog")`, note, file); got != true {
		t.Errorf("inFolder = %v", got)
	}
	if got := evalWith(t, `file.hasProperty("status")`, note, file); got != true {
		t.Errorf("hasProperty = %v", got)
	}
	if got := evalWith(t, `file.hasLink("Other Note")`, note, file); got != true {
		t.Errorf("hasLink = %v", got)
	}
	l := evalWith(t, `file.asLink()`, note, file)
	lv, ok := l.(link)
	if !ok || lv.Path != "backlog/note.md" {
		t.Errorf("asLink = %#v", l)
	}
}

func TestFunctions_GlobalMisc(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{}}
	if got := mustEval(t, `escapeHTML("<b>")`, ctx); got != "&lt;b&gt;" {
		t.Errorf("escapeHTML = %v", got)
	}
	if got := mustEval(t, `max(1, 5, 3)`, ctx); got != 5.0 {
		t.Errorf("max = %v", got)
	}
	if got := mustEval(t, `min(1, 5, 3)`, ctx); got != 1.0 {
		t.Errorf("min = %v", got)
	}
	l := mustEval(t, `list(5)`, ctx).([]any)
	if len(l) != 1 || l[0] != 5.0 {
		t.Errorf("list(5) = %v", l)
	}
	lk := mustEval(t, `link("Some Note", "display")`, ctx).(link)
	if lk.Path != "Some Note" || lk.Display != "display" {
		t.Errorf("link() = %#v", lk)
	}
}

func TestFunctions_NotYetImplementedStubsError(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"l": []any{1.0, 2.0}}}
	for _, expr := range []string{
		`file("other.md")`,
		`link("x").asFile()`,
		`link("x").linksTo("y")`,
		`l.filter(true)`,
		`l.map(1)`,
		`l.reduce(1, 0)`,
	} {
		if _, err := EvalExpr(expr, ctx); err == nil {
			t.Errorf("%s: expected a not-yet-implemented error, got nil", expr)
		}
	}
}

func TestFunctions_DateRelative(t *testing.T) {
	ctx := &EvalContext{Note: map[string]any{"d": time.Now().Add(-2 * time.Hour)}}
	got := mustEval(t, `d.relative()`, ctx)
	s, ok := got.(string)
	if !ok || s == "" {
		t.Errorf("relative() = %v", got)
	}
}
