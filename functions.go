package sill

import (
	"fmt"
	"html"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

// callGlobal and callMethod are the function/method dispatch points for
// the evaluator, covering Obsidian's documented function reference
// (https://obsidian.md/help/bases/functions).
//
// file(), link.asFile(), and link.linksTo() need vault-wide note
// resolution — looking up some OTHER note's metadata or links — which is
// exactly what ctx.Vault provides when an EvalContext comes from
// LoadVault (see vault.go, DEC-NqI23Hxjzx). An EvalContext built
// standalone (e.g. in a test) has a nil Vault, and these three report
// that plainly rather than panicking.
//
// Not implemented: the Regular Expression type's .matches() — the docs
// document no way to construct a regexp value from within an expression
// (no regexp() global, no literal syntax), so there is nothing to call
// it on.
//
// ponytail: list.filter/map/reduce take a per-element expression, which
// needs a lambda-binding mechanism this evaluator doesn't have (what
// identifier does the expression bind to? not confirmed from the docs).
// Stubbed rather than guessed. Upgrade: special-case these three in
// eval.go's *callExpr handling (like && / || already are) to evaluate
// the argument Expr once per element against an extended EvalContext.

func callGlobal(name string, args []any, ctx *EvalContext) (any, error) {
	switch name {
	case "if":
		if len(args) < 2 {
			return nil, fmt.Errorf("if: expected at least 2 arguments, got %d", len(args))
		}
		if isTruthy(args[0]) {
			return args[1], nil
		}
		if len(args) > 2 {
			return args[2], nil
		}
		return nil, nil

	case "now":
		return time.Now(), nil

	case "today":
		y, m, d := time.Now().Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local), nil

	case "date":
		s, err := stringArg(args, 0, "date")
		if err != nil {
			return nil, err
		}
		return parseDate(s)

	case "duration":
		s, err := stringArg(args, 0, "duration")
		if err != nil {
			return nil, err
		}
		return parseDuration(s)

	case "max", "min":
		if len(args) == 0 {
			return nil, fmt.Errorf("%s: expected at least 1 argument", name)
		}
		best, ok := toNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("%s: argument 1 is not a number", name)
		}
		for i, a := range args[1:] {
			v, ok := toNumber(a)
			if !ok {
				return nil, fmt.Errorf("%s: argument %d is not a number", name, i+2)
			}
			if (name == "max" && v > best) || (name == "min" && v < best) {
				best = v
			}
		}
		return best, nil

	case "escapeHTML":
		s, err := stringArg(args, 0, "escapeHTML")
		if err != nil {
			return nil, err
		}
		return html.EscapeString(s), nil

	case "html", "image", "icon":
		// Rendering-only wrappers — sill evaluates filters/formulas, it
		// doesn't render views (see CTX-jOPDBK9Zwu's non-goals), so these
		// pass their argument straight through.
		if len(args) == 0 {
			return nil, fmt.Errorf("%s: expected 1 argument", name)
		}
		return args[0], nil

	case "link":
		if len(args) == 0 {
			return nil, fmt.Errorf("link: expected at least 1 argument")
		}
		path := toDisplayString(args[0])
		display := ""
		if len(args) > 1 {
			display = toDisplayString(args[1])
		}
		return link{Path: path, Display: display}, nil

	case "list":
		if len(args) == 0 {
			return nil, fmt.Errorf("list: expected 1 argument")
		}
		return []any{args[0]}, nil

	case "number":
		if len(args) == 0 {
			return nil, fmt.Errorf("number: expected 1 argument")
		}
		return coerceNumber(args[0])

	case "random":
		return rand.Float64(), nil

	case "file":
		p, err := stringArg(args, 0, "file")
		if err != nil {
			return nil, err
		}
		if ctx.Vault == nil {
			return nil, fmt.Errorf("file(): no vault loaded — call via sill.Query/Lint, not a standalone EvalContext")
		}
		fr, ok := ctx.Vault.Resolve(p)
		if !ok {
			return nil, nil // unresolved path: nil, same as a missing property
		}
		return fr, nil

	default:
		return nil, fmt.Errorf("unknown function %q", name)
	}
}

func callMethod(target any, name string, args []any, ctx *EvalContext) (any, error) {
	// Any-type methods apply to every value, evaluated before the
	// type-specific dispatch below so no type needs to redeclare them.
	switch name {
	case "isTruthy":
		return isTruthy(target), nil
	case "isType":
		want, err := stringArg(args, 0, "isType")
		if err != nil {
			return nil, err
		}
		return typeName(target) == want, nil
	case "toString":
		return toDisplayString(target), nil
	}

	switch t := target.(type) {
	case nil:
		if name == "isEmpty" {
			return true, nil
		}

	case time.Time:
		return dateMethod(t, name, args)

	case duration:
		// No documented duration methods beyond what arithmetic already
		// produces; dateMethod/numberMethod don't apply.

	case float64:
		return numberMethod(t, name, args)

	case string:
		return stringMethod(t, name, args)

	case []any:
		return listMethod(t, name, args)

	case map[string]any:
		return objectMethod(t, name, args)

	case link:
		switch name {
		case "asFile":
			if ctx.Vault == nil {
				return nil, fmt.Errorf("link.asFile(): no vault loaded — call via sill.Query/Lint, not a standalone EvalContext")
			}
			fr, ok := ctx.Vault.Resolve(t.Path)
			if !ok {
				return nil, nil
			}
			return fr, nil
		case "linksTo":
			if len(args) == 0 {
				return nil, fmt.Errorf("linksTo: expected 1 argument")
			}
			if ctx.Vault == nil {
				return nil, fmt.Errorf("link.linksTo(): no vault loaded — call via sill.Query/Lint, not a standalone EvalContext")
			}
			// t.Path and the argument are both unresolved queries (the
			// raw string link() was given, or a bare property value) —
			// the graph is keyed by resolved note path, so both ends
			// need resolving through the same identity Resolve uses
			// before LinksTo can match anything.
			src, ok := ctx.Vault.Resolve(t.Path)
			if !ok {
				return false, nil
			}
			dst, ok := ctx.Vault.Resolve(linkTarget(args[0]))
			if !ok {
				return false, nil
			}
			return ctx.Vault.LinksTo(src.Path, dst.Path), nil
		}

	case fileRoot:
		return fileMethod(ctx.Note, ctx.File, ctx.Vault, name, args)

	case fileRef:
		return fileMethod(t.Note, t.File, ctx.Vault, name, args)

	case noteRoot:
		return nil, fmt.Errorf("note has no method %q", name)

	case formulaRoot:
		return nil, fmt.Errorf("formula has no method %q", name)
	}

	return nil, fmt.Errorf("%s has no method %q", typeName(target), name)
}

// ── Date ─────────────────────────────────────────────────────────────────

func dateMethod(t time.Time, name string, args []any) (any, error) {
	switch name {
	case "date":
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location()), nil
	case "format":
		layout, err := stringArg(args, 0, "format")
		if err != nil {
			return nil, err
		}
		return t.Format(momentToGoLayout(layout)), nil
	case "time":
		return t.Format("15:04:05"), nil
	case "relative":
		return relativeTime(t), nil
	case "isEmpty":
		return false, nil // a Date value, once present, is never "empty"
	}
	return nil, fmt.Errorf("date has no method %q", name)
}

func relativeTime(t time.Time) string {
	d := time.Since(t)
	future := d < 0
	if future {
		d = -d
	}
	var s string
	switch {
	case d < time.Minute:
		s = "a few seconds"
	case d < time.Hour:
		n := int(d / time.Minute)
		s = fmt.Sprintf("%d minute%s", n, plural(n))
	case d < 24*time.Hour:
		n := int(d / time.Hour)
		s = fmt.Sprintf("%d hour%s", n, plural(n))
	case d < 30*24*time.Hour:
		n := int(d / (24 * time.Hour))
		s = fmt.Sprintf("%d day%s", n, plural(n))
	default:
		n := int(d / (30 * 24 * time.Hour))
		s = fmt.Sprintf("%d month%s", n, plural(n))
	}
	if future {
		return "in " + s
	}
	return s + " ago"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ── Number ───────────────────────────────────────────────────────────────

func numberMethod(n float64, name string, args []any) (any, error) {
	switch name {
	case "abs":
		return math.Abs(n), nil
	case "ceil":
		return math.Ceil(n), nil
	case "floor":
		return math.Floor(n), nil
	case "isEmpty":
		return false, nil // present numbers are never "empty"
	case "round":
		digits := 0
		if len(args) > 0 {
			p, ok := toNumber(args[0])
			if !ok {
				return nil, fmt.Errorf("round: digits is not a number")
			}
			digits = int(p)
		}
		mul := math.Pow(10, float64(digits))
		return math.Round(n*mul) / mul, nil
	case "toFixed":
		prec := 0
		if len(args) > 0 {
			p, ok := toNumber(args[0])
			if !ok {
				return nil, fmt.Errorf("toFixed: precision is not a number")
			}
			prec = int(p)
		}
		return strconv.FormatFloat(n, 'f', prec, 64), nil
	}
	return nil, fmt.Errorf("number has no method %q", name)
}

func coerceNumber(v any) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case bool:
		if t {
			return 1, nil
		}
		return 0, nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return 0, fmt.Errorf("number: cannot convert %q to a number", t)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("number: cannot convert %s to a number", typeName(v))
	}
}

// ── String ───────────────────────────────────────────────────────────────

func stringMethod(s string, name string, args []any) (any, error) {
	switch name {
	case "contains":
		v, err := stringArg(args, 0, "contains")
		if err != nil {
			return nil, err
		}
		return strings.Contains(s, v), nil
	case "containsAll":
		for i := range args {
			v, err := stringArg(args, i, "containsAll")
			if err != nil {
				return nil, err
			}
			if !strings.Contains(s, v) {
				return false, nil
			}
		}
		return true, nil
	case "containsAny":
		for i := range args {
			v, err := stringArg(args, i, "containsAny")
			if err != nil {
				return nil, err
			}
			if strings.Contains(s, v) {
				return true, nil
			}
		}
		return false, nil
	case "endsWith":
		v, err := stringArg(args, 0, "endsWith")
		if err != nil {
			return nil, err
		}
		return strings.HasSuffix(s, v), nil
	case "isEmpty":
		return s == "", nil
	case "lower":
		return strings.ToLower(s), nil
	case "replace":
		pattern, err := stringArg(args, 0, "replace")
		if err != nil {
			return nil, err
		}
		repl, err := stringArg(args, 1, "replace")
		if err != nil {
			return nil, err
		}
		// ponytail: literal substring replace only — no regex-literal
		// syntax was found documented, so "supports capture groups" (per
		// the docs) is not reachable from sill's expression language yet.
		return strings.ReplaceAll(s, pattern, repl), nil
	case "repeat":
		n, err := intArg(args, 0, "repeat")
		if err != nil {
			return nil, err
		}
		return strings.Repeat(s, n), nil
	case "reverse":
		r := []rune(s)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return string(r), nil
	case "slice":
		start, end, err := sliceArgs(args, len([]rune(s)))
		if err != nil {
			return nil, err
		}
		return string([]rune(s)[start:end]), nil
	case "split":
		sep, err := stringArg(args, 0, "split")
		if err != nil {
			return nil, err
		}
		var parts []string
		if len(args) > 1 {
			n, err := intArg(args, 1, "split")
			if err != nil {
				return nil, err
			}
			parts = strings.SplitN(s, sep, n)
		} else {
			parts = strings.Split(s, sep)
		}
		out := make([]any, len(parts))
		for i, p := range parts {
			out[i] = p
		}
		return out, nil
	case "startsWith":
		v, err := stringArg(args, 0, "startsWith")
		if err != nil {
			return nil, err
		}
		return strings.HasPrefix(s, v), nil
	case "title":
		return titleCase(s), nil
	case "trim":
		return strings.TrimSpace(s), nil
	}
	return nil, fmt.Errorf("string has no method %q", name)
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		r := []rune(w)
		if len(r) > 0 {
			r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		}
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// ── List ─────────────────────────────────────────────────────────────────

func listMethod(l []any, name string, args []any) (any, error) {
	switch name {
	case "contains":
		if len(args) == 0 {
			return nil, fmt.Errorf("contains: expected 1 argument")
		}
		for _, e := range l {
			if valuesEqual(e, args[0]) {
				return true, nil
			}
		}
		return false, nil
	case "containsAll":
		for _, want := range args {
			found := false
			for _, e := range l {
				if valuesEqual(e, want) {
					found = true
					break
				}
			}
			if !found {
				return false, nil
			}
		}
		return true, nil
	case "containsAny":
		for _, want := range args {
			for _, e := range l {
				if valuesEqual(e, want) {
					return true, nil
				}
			}
		}
		return false, nil
	case "flat":
		var out []any
		for _, e := range l {
			if sub, ok := e.([]any); ok {
				out = append(out, sub...)
			} else {
				out = append(out, e)
			}
		}
		return out, nil
	case "isEmpty":
		return len(l) == 0, nil
	case "join":
		sep := ""
		if len(args) > 0 {
			s, err := stringArg(args, 0, "join")
			if err != nil {
				return nil, err
			}
			sep = s
		}
		parts := make([]string, len(l))
		for i, e := range l {
			parts[i] = toDisplayString(e)
		}
		return strings.Join(parts, sep), nil
	case "reverse":
		out := make([]any, len(l))
		for i, e := range l {
			out[len(l)-1-i] = e
		}
		return out, nil
	case "slice":
		start, end, err := sliceArgs(args, len(l))
		if err != nil {
			return nil, err
		}
		out := make([]any, end-start)
		copy(out, l[start:end])
		return out, nil
	case "sort":
		out := make([]any, len(l))
		copy(out, l)
		sort.SliceStable(out, func(i, j int) bool {
			lt, err := compareValues("<", out[i], out[j])
			return err == nil && lt
		})
		return out, nil
	case "unique":
		var out []any
		for _, e := range l {
			dup := false
			for _, seen := range out {
				if valuesEqual(e, seen) {
					dup = true
					break
				}
			}
			if !dup {
				out = append(out, e)
			}
		}
		return out, nil
	case "filter", "map", "reduce":
		return nil, fmt.Errorf("list.%s(): not yet implemented — needs a per-element lambda-binding mechanism this evaluator doesn't have yet", name)
	}
	return nil, fmt.Errorf("list has no method %q", name)
}

// ── Object ───────────────────────────────────────────────────────────────

func objectMethod(o map[string]any, name string, args []any) (any, error) {
	switch name {
	case "isEmpty":
		return len(o) == 0, nil
	case "keys":
		out := make([]any, 0, len(o))
		for k := range o {
			out = append(out, k)
		}
		return out, nil
	case "values":
		out := make([]any, 0, len(o))
		for _, v := range o {
			out = append(out, v)
		}
		return out, nil
	}
	return nil, fmt.Errorf("object has no method %q", name)
}

// ── File ─────────────────────────────────────────────────────────────────

// fileMethod implements file.* methods against note/file — the current
// note's own data for fileRoot, or an arbitrary resolved note's data for
// fileRef (see callMethod). vault is only needed by methods that resolve
// other notes in turn; none of these do today, but it's threaded through
// for symmetry with link's asFile/linksTo.
func fileMethod(note map[string]any, file FileMeta, vault *Vault, name string, args []any) (any, error) {
	switch name {
	case "asLink":
		display := file.Name
		if len(args) > 0 {
			d, err := stringArg(args, 0, "asLink")
			if err != nil {
				return nil, err
			}
			display = d
		}
		return link{Path: file.Path, Display: display}, nil
	case "hasLink":
		if len(args) == 0 {
			return nil, fmt.Errorf("hasLink: expected 1 argument")
		}
		target := linkTarget(args[0])
		for _, l := range file.Links {
			if l == target {
				return true, nil
			}
		}
		return false, nil
	case "hasProperty":
		prop, err := stringArg(args, 0, "hasProperty")
		if err != nil {
			return nil, err
		}
		_, ok := note[prop]
		return ok, nil
	case "hasTag":
		for i := range args {
			want, err := stringArg(args, i, "hasTag")
			if err != nil {
				return nil, err
			}
			for _, tag := range file.Tags {
				if tag == want {
					return true, nil
				}
			}
		}
		return false, nil
	case "inFolder":
		folder, err := stringArg(args, 0, "inFolder")
		if err != nil {
			return nil, err
		}
		folder = strings.TrimSuffix(folder, "/")
		return file.Folder == folder || strings.HasPrefix(file.Folder, folder+"/"), nil
	}
	return nil, fmt.Errorf("file has no method %q", name)
}

func linkTarget(v any) string {
	switch t := v.(type) {
	case link:
		return t.Path
	case fileRef:
		return t.Path
	default:
		return toDisplayString(v)
	}
}

// ── Argument helpers ─────────────────────────────────────────────────────

func stringArg(args []any, i int, fn string) (string, error) {
	if i >= len(args) {
		return "", fmt.Errorf("%s: expected argument %d", fn, i+1)
	}
	s, ok := args[i].(string)
	if !ok {
		return "", fmt.Errorf("%s: argument %d is not a string", fn, i+1)
	}
	return s, nil
}

func intArg(args []any, i int, fn string) (int, error) {
	if i >= len(args) {
		return 0, fmt.Errorf("%s: expected argument %d", fn, i+1)
	}
	n, ok := toNumber(args[i])
	if !ok {
		return 0, fmt.Errorf("%s: argument %d is not a number", fn, i+1)
	}
	return int(n), nil
}

// sliceArgs normalizes (start, end?) arguments against length, supporting
// negative indices (counted from the end) and clamping into [0, length].
func sliceArgs(args []any, length int) (start, end int, err error) {
	start = 0
	end = length
	if len(args) > 0 {
		start, err = intArg(args, 0, "slice")
		if err != nil {
			return 0, 0, err
		}
	}
	if len(args) > 1 {
		end, err = intArg(args, 1, "slice")
		if err != nil {
			return 0, 0, err
		}
	}
	start = clampIndex(start, length)
	end = clampIndex(end, length)
	if end < start {
		end = start
	}
	return start, end, nil
}

func clampIndex(i, length int) int {
	if i < 0 {
		i += length
	}
	if i < 0 {
		return 0
	}
	if i > length {
		return length
	}
	return i
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
