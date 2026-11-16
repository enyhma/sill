package sill

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// duration is a parsed Bases duration value. Years/Months are kept
// separate from Clock because they're calendar-length (variable), not a
// fixed span — time.Time.AddDate handles that correctly, time.Duration
// cannot represent it at all.
type duration struct {
	Years, Months, Days int
	Clock               time.Duration
}

func (d duration) negate() duration {
	return duration{Years: -d.Years, Months: -d.Months, Days: -d.Days, Clock: -d.Clock}
}

func (d duration) addTo(t time.Time) time.Time {
	return t.AddDate(d.Years, d.Months, d.Days).Add(d.Clock)
}

// formatDuration renders a duration for string contexts (toString(),
// string concatenation). Not a documented Bases format — there is no
// duration.format() in the spec — just a readable fallback.
func formatDuration(d duration) string {
	var parts []string
	if d.Years != 0 {
		parts = append(parts, fmt.Sprintf("%dy", d.Years))
	}
	if d.Months != 0 {
		parts = append(parts, fmt.Sprintf("%dM", d.Months))
	}
	if d.Days != 0 {
		parts = append(parts, fmt.Sprintf("%dd", d.Days))
	}
	if d.Clock != 0 {
		parts = append(parts, d.Clock.String())
	}
	if len(parts) == 0 {
		return "0s"
	}
	return strings.Join(parts, " ")
}

var durationRe = regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*(years?|months?|weeks?|days?|hours?|minutes?|seconds?|[yMwdhms])$`)

// parseDuration parses a Bases duration string, e.g. "1M", "2h", "5 days".
// Unit letters are case-sensitive where it matters: "M" is months, "m" is
// minutes.
func parseDuration(s string) (duration, error) {
	m := durationRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return duration{}, fmt.Errorf("invalid duration %q", s)
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return duration{}, err
	}
	switch m[2] {
	case "y", "year", "years":
		return duration{Years: int(n)}, nil
	case "M", "month", "months":
		return duration{Months: int(n)}, nil
	case "w", "week", "weeks":
		return duration{Days: int(n) * 7}, nil
	case "d", "day", "days":
		return duration{Days: int(n)}, nil
	case "h", "hour", "hours":
		return duration{Clock: time.Duration(n * float64(time.Hour))}, nil
	case "m", "minute", "minutes":
		return duration{Clock: time.Duration(n * float64(time.Minute))}, nil
	case "s", "second", "seconds":
		return duration{Clock: time.Duration(n * float64(time.Second))}, nil
	}
	return duration{}, fmt.Errorf("invalid duration unit %q", m[2])
}

// coerceDuration accepts either an already-parsed duration or a raw
// duration string (the spec allows writing `date + "1M"` directly,
// without an explicit duration() call).
func coerceDuration(v any) (duration, error) {
	switch t := v.(type) {
	case duration:
		return t, nil
	case string:
		return parseDuration(t)
	default:
		return duration{}, fmt.Errorf("cannot use %T as a duration", v)
	}
}

var dateLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseDate parses a Bases date() string.
func parseDate(s string) (time.Time, error) {
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q", s)
}

// momentPairs maps Moment.js format tokens to Go's reference-time layout,
// longest tokens first so e.g. "YYYY" is substituted before "YY".
//
// ponytail: covers the common date/time tokens only (Y/M/D/H/m/s family),
// not Moment's full token set (dddd, Do, A/a, etc.). Extend when a real
// .format() call needs a token not listed here.
var momentPairs = []struct{ from, to string }{
	{"YYYY", "2006"},
	{"YY", "06"},
	{"MM", "01"},
	{"DD", "02"},
	{"HH", "15"},
	{"hh", "03"},
	{"mm", "04"},
	{"ss", "05"},
	{"M", "1"},
	{"D", "2"},
}

func momentToGoLayout(format string) string {
	for _, p := range momentPairs {
		format = strings.ReplaceAll(format, p.from, p.to)
	}
	return format
}
