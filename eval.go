package sill

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

// FileMeta is the subset of file.* metadata a note carries, independent
// of its frontmatter properties.
type FileMeta struct {
	Name, Basename, Path, Folder, Ext string
	Size                              int64
	CTime, MTime                      time.Time
	Tags, Links                       []string
}

// EvalContext is what a filter/formula expression evaluates against: one
// note's frontmatter properties, its file metadata, the base's own
// formula definitions (resolved lazily and cached on first reference),
// and — when evaluating as part of a loaded vault — a Vault for
// resolving references to OTHER notes (file(), link.asFile(),
// link.linksTo()). Vault is nil when an EvalContext is built standalone
// (e.g. in tests); those three functions then report that there's no
// vault to resolve against, same shape as any other input error.
type EvalContext struct {
	Note     map[string]any
	File     FileMeta
	Formulas map[string]string
	Vault    *Vault

	formulaCache map[string]any
}

// Vault is the cross-note capability a loaded set of notes provides.
// Kept as plain funcs (not a hearth type) so eval.go/functions.go have no
// hearth dependency — vault.go is the one file that builds a real Vault,
// backed by hearth.BuildGraph and hearth.Match (see DEC-NqI23Hxjzx).
type Vault struct {
	// LinksTo reports whether the note at path a's wikilinks resolve to
	// path b — backed by hearth's backlink graph.
	LinksTo func(a, b string) bool
	// Resolve looks up a note by path, name, or wikilink-style query,
	// returning (zero, false) when it doesn't resolve to exactly one note.
	Resolve func(query string) (fileRef, bool)
}

// root markers: evaluating the bare identifiers "note"/"file"/"formula"
// yields one of these, which a following Member/Call dispatches on.
type noteRoot struct{}
type fileRoot struct{}
type formulaRoot struct{}

// fileRef is a resolved reference to a specific note in the vault — what
// the global file() function and link.asFile() return. Unlike fileRoot
// (the current note's implicit "file" root, read through ctx.Note/
// ctx.File), it carries its own note properties and file metadata so
// methods and fields work on it directly, independent of "the current
// note".
type fileRef struct {
	Path string
	Note map[string]any
	File FileMeta
}

// link is the value produced by the global link() function and by
// file.asLink() — a lightweight path+display wrapper. Resolving it (via
// Vault.Resolve) or querying what it links to (via Vault.LinksTo) needs
// a loaded vault; without one, both report that plainly.
type link struct {
	Path    string
	Display string
}

// EvalExpr parses and evaluates a single expression string — a filter
// statement or a formula body — against ctx.
func EvalExpr(expr string, ctx *EvalContext) (any, error) {
	ast, err := parseExpr(expr)
	if err != nil {
		return nil, fmt.Errorf("parsing %q: %w", expr, err)
	}
	return eval(ast, ctx)
}

// EvalFilter evaluates a parsed Filter tree against ctx. A nil Filter
// (no filters: key) passes everything.
func EvalFilter(f *Filter, ctx *EvalContext) (bool, error) {
	if f == nil {
		return true, nil
	}
	switch {
	case f.And != nil:
		for _, sub := range f.And {
			ok, err := EvalFilter(&sub, ctx)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	case f.Or != nil:
		for _, sub := range f.Or {
			ok, err := EvalFilter(&sub, ctx)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	case f.Not != nil:
		// not: holds a list; read as NOT(AND(list)) — true as soon as any
		// sub-filter fails, matching the single-item lists shown in the
		// documented examples.
		for _, sub := range f.Not {
			ok, err := EvalFilter(&sub, ctx)
			if err != nil {
				return false, err
			}
			if !ok {
				return true, nil
			}
		}
		return false, nil
	default:
		v, err := EvalExpr(f.Statement, ctx)
		if err != nil {
			return false, err
		}
		return isTruthy(v), nil
	}
}

func (ctx *EvalContext) evalFormula(name string) (any, error) {
	if v, ok := ctx.formulaCache[name]; ok {
		return v, nil
	}
	expr, ok := ctx.Formulas[name]
	if !ok {
		return nil, fmt.Errorf("undefined formula %q", name)
	}
	v, err := EvalExpr(expr, ctx)
	if err != nil {
		return nil, fmt.Errorf("formula %q: %w", name, err)
	}
	if ctx.formulaCache == nil {
		ctx.formulaCache = make(map[string]any)
	}
	ctx.formulaCache[name] = v
	return v, nil
}

func eval(e Expr, ctx *EvalContext) (any, error) {
	switch n := e.(type) {
	case *litExpr:
		return n.Value, nil

	case *identExpr:
		switch n.Name {
		case "note":
			return noteRoot{}, nil
		case "file":
			return fileRoot{}, nil
		case "formula":
			return formulaRoot{}, nil
		default:
			return ctx.Note[n.Name], nil // bare shorthand for note.<name>
		}

	case *memberExpr:
		t, err := eval(n.Target, ctx)
		if err != nil {
			return nil, err
		}
		return evalMember(t, n.Name, ctx)

	case *indexExpr:
		t, err := eval(n.Target, ctx)
		if err != nil {
			return nil, err
		}
		k, err := eval(n.Key, ctx)
		if err != nil {
			return nil, err
		}
		key, _ := k.(string)
		return evalMember(t, key, ctx)

	case *callExpr:
		args := make([]any, len(n.Args))
		for i, a := range n.Args {
			v, err := eval(a, ctx)
			if err != nil {
				return nil, err
			}
			args[i] = v
		}
		if n.Target == nil {
			return callGlobal(n.Name, args, ctx)
		}
		t, err := eval(n.Target, ctx)
		if err != nil {
			return nil, err
		}
		return callMethod(t, n.Name, args, ctx)

	case *unaryExpr:
		x, err := eval(n.X, ctx)
		if err != nil {
			return nil, err
		}
		switch n.Op {
		case "!":
			return !isTruthy(x), nil
		case "-":
			f, ok := toNumber(x)
			if !ok {
				return nil, fmt.Errorf("cannot negate %T", x)
			}
			return -f, nil
		}
		return nil, fmt.Errorf("unknown unary operator %q", n.Op)

	case *binaryExpr:
		if n.Op == "&&" || n.Op == "||" {
			x, err := eval(n.X, ctx)
			if err != nil {
				return nil, err
			}
			xt := isTruthy(x)
			if n.Op == "&&" && !xt {
				return false, nil
			}
			if n.Op == "||" && xt {
				return true, nil
			}
			y, err := eval(n.Y, ctx)
			if err != nil {
				return nil, err
			}
			return isTruthy(y), nil
		}
		x, err := eval(n.X, ctx)
		if err != nil {
			return nil, err
		}
		y, err := eval(n.Y, ctx)
		if err != nil {
			return nil, err
		}
		return applyBinary(n.Op, x, y)

	default:
		return nil, fmt.Errorf("unhandled expression node %T", e)
	}
}

func evalMember(target any, name string, ctx *EvalContext) (any, error) {
	switch t := target.(type) {
	case noteRoot:
		return ctx.Note[name], nil
	case fileRoot:
		if name == "properties" {
			return ctx.Note, nil
		}
		return evalFileField(ctx.File, name)
	case fileRef:
		if name == "properties" {
			return t.Note, nil
		}
		return evalFileField(t.File, name)
	case formulaRoot:
		return ctx.evalFormula(name)
	case map[string]any:
		return t[name], nil
	default:
		return evalField(target, name)
	}
}

func evalFileField(f FileMeta, name string) (any, error) {
	switch name {
	case "name":
		return f.Name, nil
	case "basename":
		return f.Basename, nil
	case "path":
		return f.Path, nil
	case "folder":
		return f.Folder, nil
	case "ext":
		return f.Ext, nil
	case "size":
		return float64(f.Size), nil
	case "ctime":
		return f.CTime, nil
	case "mtime":
		return f.MTime, nil
	case "tags":
		return toAnySlice(f.Tags), nil
	case "links":
		return toAnySlice(f.Links), nil
	default:
		return nil, fmt.Errorf("file has no field %q", name)
	}
}

// evalField resolves a no-parens member access (a type's documented
// "fields", e.g. date.year, string.length) on an already-evaluated value.
func evalField(target any, name string) (any, error) {
	switch t := target.(type) {
	case time.Time:
		switch name {
		case "year":
			return float64(t.Year()), nil
		case "month":
			return float64(t.Month()), nil
		case "day":
			return float64(t.Day()), nil
		case "hour":
			return float64(t.Hour()), nil
		case "minute":
			return float64(t.Minute()), nil
		case "second":
			return float64(t.Second()), nil
		case "millisecond":
			return float64(t.Nanosecond() / 1e6), nil
		}
	case string:
		if name == "length" {
			return float64(len([]rune(t))), nil
		}
	case []any:
		if name == "length" {
			return float64(len(t)), nil
		}
	}
	return nil, fmt.Errorf("%s has no field %q (not yet implemented — see CAP-E7loYprwhZ)", typeName(target), name)
}

func applyBinary(op string, x, y any) (any, error) {
	switch op {
	case "+", "-":
		if xd, ok := x.(time.Time); ok {
			if yd, ok := y.(time.Time); ok && op == "-" {
				return duration{Clock: xd.Sub(yd)}, nil
			}
			d, err := coerceDuration(y)
			if err != nil {
				return nil, err
			}
			if op == "-" {
				d = d.negate()
			}
			return d.addTo(xd), nil
		}
		if op == "+" {
			if xs, ok := x.(string); ok {
				return xs + toDisplayString(y), nil
			}
			if ys, ok := y.(string); ok {
				return toDisplayString(x) + ys, nil
			}
		}
		xf, xok := toNumber(x)
		yf, yok := toNumber(y)
		if !xok || !yok {
			return nil, fmt.Errorf("cannot apply %q to %s and %s", op, typeName(x), typeName(y))
		}
		if op == "+" {
			return xf + yf, nil
		}
		return xf - yf, nil

	case "*", "/", "%":
		xf, xok := toNumber(x)
		yf, yok := toNumber(y)
		if !xok || !yok {
			return nil, fmt.Errorf("cannot apply %q to %s and %s", op, typeName(x), typeName(y))
		}
		switch op {
		case "*":
			return xf * yf, nil
		case "/":
			return xf / yf, nil
		default:
			return math.Mod(xf, yf), nil
		}

	case "==":
		return valuesEqual(x, y), nil
	case "!=":
		return !valuesEqual(x, y), nil
	case "<", "<=", ">", ">=":
		return compareValues(op, x, y)
	default:
		return nil, fmt.Errorf("unknown binary operator %q", op)
	}
}

func valuesEqual(x, y any) bool {
	if xf, xok := toNumber(x); xok {
		if yf, yok := toNumber(y); yok {
			return xf == yf
		}
	}
	return reflect.DeepEqual(x, y)
}

func compareValues(op string, x, y any) (bool, error) {
	cmp := func(lt, eq bool) bool {
		switch op {
		case "<":
			return lt
		case "<=":
			return lt || eq
		case ">":
			return !lt && !eq
		default: // ">="
			return !lt
		}
	}
	if xf, xok := toNumber(x); xok {
		if yf, yok := toNumber(y); yok {
			return cmp(xf < yf, xf == yf), nil
		}
	}
	if xs, ok := x.(string); ok {
		if ys, ok := y.(string); ok {
			return cmp(xs < ys, xs == ys), nil
		}
	}
	if xt, ok := x.(time.Time); ok {
		if yt, ok := y.(time.Time); ok {
			return cmp(xt.Before(yt), xt.Equal(yt)), nil
		}
	}
	return false, fmt.Errorf("cannot compare %s and %s", typeName(x), typeName(y))
}

func isTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case []any:
		return len(t) > 0
	default:
		return true
	}
}

func toNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func toDisplayString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return formatFloat(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	case duration:
		return formatDuration(t)
	case link:
		return t.Path
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = toDisplayString(e)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	case time.Time:
		return "date"
	case duration:
		return "duration"
	case []any:
		return "list"
	case map[string]any:
		return "object"
	case link:
		return "link"
	case noteRoot:
		return "note"
	case fileRoot, fileRef:
		return "file"
	case formulaRoot:
		return "formula"
	default:
		return fmt.Sprintf("%T", v)
	}
}
