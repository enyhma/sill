package sill

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// callGlobal and callMethod are the function/method dispatch points for
// the evaluator. Only the functions CAP-MB8sZOUUno's own acceptance
// criteria exercise (if, date arithmetic, basic number formatting) are
// implemented here — the full documented catalog is CAP-E7loYprwhZ's
// capability, built on this same dispatch mechanism.

func callGlobal(name string, args []any) (any, error) {
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

	default:
		return nil, fmt.Errorf("unknown function %q (not yet implemented — see CAP-E7loYprwhZ)", name)
	}
}

func callMethod(target any, name string, args []any) (any, error) {
	switch t := target.(type) {
	case time.Time:
		switch name {
		case "format":
			layout, err := stringArg(args, 0, "format")
			if err != nil {
				return nil, err
			}
			return t.Format(momentToGoLayout(layout)), nil
		case "date":
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, t.Location()), nil
		}

	case float64:
		switch name {
		case "toFixed":
			prec := 0
			if len(args) > 0 {
				p, ok := toNumber(args[0])
				if !ok {
					return nil, fmt.Errorf("toFixed: precision is not a number")
				}
				prec = int(p)
			}
			return strconv.FormatFloat(t, 'f', prec, 64), nil
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
			return math.Round(t*mul) / mul, nil
		case "abs":
			return math.Abs(t), nil
		case "ceil":
			return math.Ceil(t), nil
		case "floor":
			return math.Floor(t), nil
		}

	case noteRoot:
		return nil, fmt.Errorf("note has no method %q", name)
	case fileRoot, formulaRoot:
		return nil, fmt.Errorf("%s has no method %q (not yet implemented — see CAP-E7loYprwhZ)", typeName(target), name)
	}

	return nil, fmt.Errorf("%s has no method %q (not yet implemented — see CAP-E7loYprwhZ)", typeName(target), name)
}

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

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
