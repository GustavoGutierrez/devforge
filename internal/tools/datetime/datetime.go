// Package datetime implements MCP tools for date and time operations.
// Tools: time_convert, time_diff, time_cron, time_date_range.
package datetime

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// errResult returns a JSON-encoded error response.
func errResult(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

// resultJSON marshals v to JSON or returns an error JSON.
func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errResult("marshal failed: " + err.Error())
	}
	return string(b)
}

// humanLayout is the readable timestamp format used for "human" mode.
const humanLayout = "Jan 2, 2006 15:04:05 MST"

// supportedFormats lists the auto-detection order for time parsing.
var supportedFormats = []struct {
	name   string
	layout string
}{
	{"rfc3339", time.RFC3339Nano},
	{"rfc3339", time.RFC3339},
	{"iso8601", "2006-01-02T15:04:05Z07:00"},
	{"iso8601", "2006-01-02T15:04:05"},
	{"iso8601", "2006-01-02 15:04:05"},
	{"iso8601", "2006-01-02"},
	{"human", humanLayout},
	{"human", "January 2, 2006 15:04:05 MST"},
	{"human", "January 2, 2006"},
	{"human", "Jan 2 2006"},
	{"human", "02 Jan 2006"},
}

// ─── time_convert ────────────────────────────────────────────────────────────

// TimeConvertInput is the input schema for the time_convert tool.
type TimeConvertInput struct {
	Input      string `json:"input"`
	FromFormat string `json:"from_format"`
	ToFormat   string `json:"to_format"`
	Timezone   string `json:"timezone"`
}

// TimeConvertOutput is the output schema for the time_convert tool.
type TimeConvertOutput struct {
	Result     string `json:"result"`
	FromFormat string `json:"from_format"`
	ToFormat   string `json:"to_format"`
	Timezone   string `json:"timezone"`
}

// TimeConvert implements the time_convert MCP tool.
// It parses a timestamp in one format and converts it to another.
func TimeConvert(_ context.Context, input TimeConvertInput) string {
	if strings.TrimSpace(input.Input) == "" {
		return errResult("input is required")
	}

	// Resolve timezone
	tzName := input.Timezone
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return errResult("unknown timezone: " + tzName)
	}

	fromFmt := strings.ToLower(strings.TrimSpace(input.FromFormat))
	if fromFmt == "" {
		fromFmt = "auto"
	}
	toFmt := strings.ToLower(strings.TrimSpace(input.ToFormat))
	if toFmt == "" {
		toFmt = "rfc3339"
	}

	// Parse the input timestamp
	t, detectedFmt, parseErr := parseTimestamp(input.Input, fromFmt, loc)
	if parseErr != nil {
		return errResult("could not parse input: " + parseErr.Error())
	}

	// Apply requested output timezone
	t = t.In(loc)

	// Format the output
	result, fmtErr := formatTimestamp(t, toFmt)
	if fmtErr != nil {
		return errResult("could not format output: " + fmtErr.Error())
	}

	return resultJSON(TimeConvertOutput{
		Result:     result,
		FromFormat: detectedFmt,
		ToFormat:   toFmt,
		Timezone:   tzName,
	})
}

// parseTimestamp parses a timestamp string using the given format hint.
// For "auto", it tries all known formats.
func parseTimestamp(input, fmtHint string, loc *time.Location) (time.Time, string, error) {
	input = strings.TrimSpace(input)

	switch fmtHint {
	case "unix":
		n, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return time.Time{}, "", fmt.Errorf("unix timestamp must be an integer: %w", err)
		}
		return time.Unix(n, 0).In(loc), "unix", nil

	case "unix_ms":
		n, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return time.Time{}, "", fmt.Errorf("unix_ms timestamp must be an integer: %w", err)
		}
		return time.UnixMilli(n).In(loc), "unix_ms", nil

	case "iso8601":
		for _, sf := range supportedFormats {
			if sf.name != "iso8601" {
				continue
			}
			if t, err := time.ParseInLocation(sf.layout, input, loc); err == nil {
				return t, "iso8601", nil
			}
		}
		return time.Time{}, "", fmt.Errorf("could not parse %q as iso8601", input)

	case "rfc3339":
		if t, err := time.Parse(time.RFC3339Nano, input); err == nil {
			return t.In(loc), "rfc3339", nil
		}
		if t, err := time.Parse(time.RFC3339, input); err == nil {
			return t.In(loc), "rfc3339", nil
		}
		return time.Time{}, "", fmt.Errorf("could not parse %q as rfc3339", input)

	case "human":
		for _, sf := range supportedFormats {
			if sf.name != "human" {
				continue
			}
			if t, err := time.ParseInLocation(sf.layout, input, loc); err == nil {
				return t, "human", nil
			}
		}
		return time.Time{}, "", fmt.Errorf("could not parse %q as human format", input)

	case "auto":
		// Try unix / unix_ms first if the value looks numeric
		if n, err := strconv.ParseInt(input, 10, 64); err == nil {
			// Heuristic: if the value is > 1e12 treat as milliseconds
			if n > 1_000_000_000_000 {
				return time.UnixMilli(n).In(loc), "unix_ms", nil
			}
			return time.Unix(n, 0).In(loc), "unix", nil
		}
		// Try all string layouts
		for _, sf := range supportedFormats {
			if t, err := time.Parse(sf.layout, input); err == nil {
				return t.In(loc), sf.name, nil
			}
			if t, err := time.ParseInLocation(sf.layout, input, loc); err == nil {
				return t.In(loc), sf.name, nil
			}
		}
		return time.Time{}, "", fmt.Errorf("could not auto-detect format for %q", input)
	}

	return time.Time{}, "", fmt.Errorf("unsupported from_format %q", fmtHint)
}

// formatTimestamp formats a time value into the requested format string.
func formatTimestamp(t time.Time, toFmt string) (string, error) {
	switch toFmt {
	case "unix":
		return strconv.FormatInt(t.Unix(), 10), nil
	case "unix_ms":
		return strconv.FormatInt(t.UnixMilli(), 10), nil
	case "iso8601":
		return t.Format("2006-01-02T15:04:05Z07:00"), nil
	case "rfc3339":
		return t.Format(time.RFC3339), nil
	case "human":
		return t.Format(humanLayout), nil
	}
	return "", fmt.Errorf("unsupported to_format %q", toFmt)
}

// ─── time_diff ───────────────────────────────────────────────────────────────

// TimeDiffInput is the input schema for the time_diff tool.
type TimeDiffInput struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Unit      string `json:"unit"`
	Operation string `json:"operation"`
	Duration  string `json:"duration"`
}

// TimeDiffResult is the output for operation=diff.
type TimeDiffResult struct {
	Seconds float64 `json:"seconds"`
	Minutes float64 `json:"minutes"`
	Hours   float64 `json:"hours"`
	Days    float64 `json:"days"`
	Human   string  `json:"human"`
}

// TimeAddResult is the output for operation=add or operation=subtract.
type TimeAddResult struct {
	Result string `json:"result"`
}

// TimeDiff implements the time_diff MCP tool.
func TimeDiff(_ context.Context, input TimeDiffInput) string {
	if strings.TrimSpace(input.Start) == "" {
		return errResult("start is required")
	}

	op := strings.ToLower(strings.TrimSpace(input.Operation))
	if op == "" {
		op = "diff"
	}

	// Parse start time
	startT, _, err := parseTimestamp(input.Start, "auto", time.UTC)
	if err != nil {
		return errResult("could not parse start: " + err.Error())
	}

	switch op {
	case "add", "subtract":
		if strings.TrimSpace(input.Duration) == "" {
			return errResult("duration is required for add/subtract operations")
		}
		dur, durErr := parseDuration(input.Duration)
		if durErr != nil {
			return errResult("could not parse duration: " + durErr.Error())
		}
		var resultT time.Time
		if op == "add" {
			resultT = startT.Add(dur)
		} else {
			resultT = startT.Add(-dur)
		}
		return resultJSON(TimeAddResult{Result: resultT.UTC().Format(time.RFC3339)})

	case "diff":
		if strings.TrimSpace(input.End) == "" {
			return errResult("end is required for diff operation")
		}
		endT, _, err := parseTimestamp(input.End, "auto", time.UTC)
		if err != nil {
			return errResult("could not parse end: " + err.Error())
		}
		diff := endT.Sub(startT)
		secs := diff.Seconds()
		mins := diff.Minutes()
		hours := diff.Hours()
		days := hours / 24

		return resultJSON(TimeDiffResult{
			Seconds: math.Round(secs*1000) / 1000,
			Minutes: math.Round(mins*1000) / 1000,
			Hours:   math.Round(hours*1000) / 1000,
			Days:    math.Round(days*1000) / 1000,
			Human:   humanizeDuration(diff),
		})

	default:
		return errResult("unsupported operation: " + op + " (valid: diff, add, subtract)")
	}
}

// parseDuration parses both Go duration strings ("2h30m") and English expressions
// like "3 days", "1 week", "30 minutes".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Try standard Go duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Try English: "<n> <unit>" — e.g. "3 days", "2 weeks", "30 minutes"
	parts := strings.Fields(s)
	if len(parts) == 2 {
		n, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("could not parse duration number %q", parts[0])
		}
		unit := strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(parts[1], "s"), "."))
		switch unit {
		case "second":
			return time.Duration(n * float64(time.Second)), nil
		case "minute":
			return time.Duration(n * float64(time.Minute)), nil
		case "hour":
			return time.Duration(n * float64(time.Hour)), nil
		case "day":
			return time.Duration(n * float64(24*time.Hour)), nil
		case "week":
			return time.Duration(n * float64(7*24*time.Hour)), nil
		}
	}

	return 0, fmt.Errorf("unsupported duration format %q — use Go duration (2h30m) or English (3 days)", s)
}

// humanizeDuration converts a time.Duration to a readable string like "2 days 3 hours 10 minutes".
func humanizeDuration(d time.Duration) string {
	if d < 0 {
		return "-" + humanizeDuration(-d)
	}
	if d == 0 {
		return "0 seconds"
	}

	parts := []string{}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d day%s", days, pluralS(days)))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d hour%s", hours, pluralS(hours)))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d minute%s", minutes, pluralS(minutes)))
	}
	if seconds > 0 && days == 0 {
		parts = append(parts, fmt.Sprintf("%d second%s", seconds, pluralS(seconds)))
	}

	if len(parts) == 0 {
		return "less than 1 second"
	}
	return strings.Join(parts, " ")
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ─── time_cron ───────────────────────────────────────────────────────────────

// TimeCronInput is the input schema for the time_cron tool.
type TimeCronInput struct {
	Expression string `json:"expression"`
	Operation  string `json:"operation"`
	Count      int    `json:"count"`
	From       string `json:"from"`
}

// TimeCronDescribeOutput is the output for operation=describe.
type TimeCronDescribeOutput struct {
	Description string `json:"description"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}

// TimeCronNextOutput is the output for operation=next.
type TimeCronNextOutput struct {
	Next  []string `json:"next"`
	Count int      `json:"count"`
}

// TimeCron implements the time_cron MCP tool.
func TimeCron(_ context.Context, input TimeCronInput) string {
	if strings.TrimSpace(input.Expression) == "" {
		return errResult("expression is required")
	}

	op := strings.ToLower(strings.TrimSpace(input.Operation))
	if op == "" {
		op = "describe"
	}

	// Build parser that handles both 5-field standard and 6-field (with seconds) cron
	parser := cron.NewParser(
		cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)

	expr := strings.TrimSpace(input.Expression)

	// For 5-field expressions, prepend "0" for the seconds field so the
	// SecondOptional parser processes it correctly.
	fields := strings.Fields(expr)
	cronExpr := expr
	if len(fields) == 5 {
		cronExpr = "0 " + expr
	}

	schedule, parseErr := parser.Parse(cronExpr)

	switch op {
	case "describe":
		if parseErr != nil {
			b, _ := json.Marshal(TimeCronDescribeOutput{
				Valid: false,
				Error: parseErr.Error(),
			})
			return string(b)
		}
		desc := describeCron(expr)
		return resultJSON(TimeCronDescribeOutput{
			Description: desc,
			Valid:       true,
		})

	case "next":
		if parseErr != nil {
			return errResult("invalid cron expression: " + parseErr.Error())
		}

		count := input.Count
		if count <= 0 {
			count = 5
		}

		var fromT time.Time
		if strings.TrimSpace(input.From) != "" {
			t, _, err := parseTimestamp(input.From, "auto", time.UTC)
			if err != nil {
				return errResult("could not parse from: " + err.Error())
			}
			fromT = t
		} else {
			fromT = time.Now().UTC()
		}

		nexts := make([]string, 0, count)
		current := fromT
		for i := 0; i < count; i++ {
			current = schedule.Next(current)
			if current.IsZero() {
				break
			}
			nexts = append(nexts, current.UTC().Format(time.RFC3339))
		}

		return resultJSON(TimeCronNextOutput{
			Next:  nexts,
			Count: len(nexts),
		})

	default:
		return errResult("unsupported operation: " + op + " (valid: describe, next)")
	}
}

// describeCron generates a human-readable English description of a cron expression.
// Supports 5-field (min hour dom month dow) and 6-field (sec min hour dom month dow).
func describeCron(expr string) string {
	fields := strings.Fields(expr)

	var secF, minF, hourF, domF, monthF, dowF string

	switch len(fields) {
	case 5:
		minF, hourF, domF, monthF, dowF = fields[0], fields[1], fields[2], fields[3], fields[4]
		secF = "0"
	case 6:
		secF, minF, hourF, domF, monthF, dowF = fields[0], fields[1], fields[2], fields[3], fields[4], fields[5]
	default:
		return "Custom cron schedule"
	}

	var sb strings.Builder

	// Build time part
	timePart := describeTime(secF, minF, hourF)
	sb.WriteString(timePart)

	// Day-of-week
	if dowF != "*" && dowF != "?" {
		sb.WriteString(", on " + describeDow(dowF))
	}

	// Day-of-month
	if domF != "*" && domF != "?" {
		sb.WriteString(", on day " + domF + " of the month")
	}

	// Month
	if monthF != "*" && monthF != "?" {
		sb.WriteString(", in " + describeMonth(monthF))
	}

	result := sb.String()
	if result == "" {
		return "Every second"
	}
	return result
}

// describeTime returns a readable description of the time portion of a cron schedule.
func describeTime(sec, min, hour string) string {
	// All wildcards → every second/minute/hour
	if sec == "*" && min == "*" && hour == "*" {
		return "Every second"
	}
	if sec == "0" && min == "*" && hour == "*" {
		return "Every minute"
	}

	// Step expressions for minute: */N
	if sec == "0" && strings.HasPrefix(min, "*/") && hour == "*" {
		n := strings.TrimPrefix(min, "*/")
		return "Every " + n + " minutes"
	}

	// Step expressions for hour: */N
	if sec == "0" && min == "0" && strings.HasPrefix(hour, "*/") {
		n := strings.TrimPrefix(hour, "*/")
		return "Every " + n + " hours"
	}

	// Specific time
	if sec == "0" && !strings.Contains(min, "*") && !strings.Contains(hour, "*") {
		h, errH := strconv.Atoi(hour)
		m, errM := strconv.Atoi(min)
		if errH == nil && errM == nil {
			ampm := "AM"
			displayH := h
			if h == 0 {
				displayH = 12
			} else if h == 12 {
				ampm = "PM"
			} else if h > 12 {
				displayH = h - 12
				ampm = "PM"
			}
			return fmt.Sprintf("At %d:%02d %s", displayH, m, ampm)
		}
	}

	// Fallback
	parts := []string{}
	if sec != "0" && sec != "*" {
		parts = append(parts, "second "+sec)
	}
	if min != "*" {
		parts = append(parts, "minute "+min)
	}
	if hour != "*" {
		parts = append(parts, "hour "+hour)
	}
	if len(parts) == 0 {
		return "Every second"
	}
	return "At " + strings.Join(parts, ", ")
}

// describeDow converts a day-of-week field to English.
func describeDow(dow string) string {
	names := map[string]string{
		"0": "Sunday", "1": "Monday", "2": "Tuesday", "3": "Wednesday",
		"4": "Thursday", "5": "Friday", "6": "Saturday", "7": "Sunday",
		"SUN": "Sunday", "MON": "Monday", "TUE": "Tuesday", "WED": "Wednesday",
		"THU": "Thursday", "FRI": "Friday", "SAT": "Saturday",
	}
	upper := strings.ToUpper(dow)
	if n, ok := names[upper]; ok {
		return n
	}
	// Ranges or lists — return as-is
	return dow
}

// describeMonth converts a month field to English.
func describeMonth(month string) string {
	names := map[string]string{
		"1": "January", "2": "February", "3": "March", "4": "April",
		"5": "May", "6": "June", "7": "July", "8": "August",
		"9": "September", "10": "October", "11": "November", "12": "December",
		"JAN": "January", "FEB": "February", "MAR": "March", "APR": "April",
		"MAY": "May", "JUN": "June", "JUL": "July", "AUG": "August",
		"SEP": "September", "OCT": "October", "NOV": "November", "DEC": "December",
	}
	upper := strings.ToUpper(month)
	if n, ok := names[upper]; ok {
		return n
	}
	return month
}

// ─── time_date_range ─────────────────────────────────────────────────────────

// TimeDateRangeInput is the input schema for the time_date_range tool.
type TimeDateRangeInput struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Step   string `json:"step"`
	Format string `json:"format"`
}

// TimeDateRangeOutput is the output schema for the time_date_range tool.
type TimeDateRangeOutput struct {
	Dates []string `json:"dates"`
	Count int      `json:"count"`
}

const maxDateRangeCount = 1000

// TimeDateRange implements the time_date_range MCP tool.
func TimeDateRange(_ context.Context, input TimeDateRangeInput) string {
	if strings.TrimSpace(input.Start) == "" {
		return errResult("start is required")
	}
	if strings.TrimSpace(input.End) == "" {
		return errResult("end is required")
	}

	step := strings.ToLower(strings.TrimSpace(input.Step))
	if step == "" {
		step = "day"
	}

	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" {
		format = "iso8601"
	}

	startT, _, err := parseTimestamp(input.Start, "auto", time.UTC)
	if err != nil {
		return errResult("could not parse start: " + err.Error())
	}
	endT, _, err := parseTimestamp(input.End, "auto", time.UTC)
	if err != nil {
		return errResult("could not parse end: " + err.Error())
	}

	// Normalize to start of day
	startT = time.Date(startT.Year(), startT.Month(), startT.Day(), 0, 0, 0, 0, time.UTC)
	endT = time.Date(endT.Year(), endT.Month(), endT.Day(), 0, 0, 0, 0, time.UTC)

	if endT.Before(startT) {
		return errResult("end must be on or after start")
	}

	// Estimate count to prevent huge allocations
	estimated := estimateDateCount(startT, endT, step)
	if estimated > maxDateRangeCount {
		return errResult(fmt.Sprintf(
			"date range would produce more than %d dates (%d estimated); narrow the range or use a larger step",
			maxDateRangeCount, estimated,
		))
	}

	var dates []string
	current := startT
	for !current.After(endT) {
		d, fmtErr := formatDate(current, format)
		if fmtErr != nil {
			return errResult(fmtErr.Error())
		}
		dates = append(dates, d)

		// Safety cap (handles edge cases where estimate is off)
		if len(dates) >= maxDateRangeCount {
			break
		}

		current = advanceDate(current, step)
	}

	return resultJSON(TimeDateRangeOutput{
		Dates: dates,
		Count: len(dates),
	})
}

// estimateDateCount estimates the number of dates in a range without generating them all.
func estimateDateCount(start, end time.Time, step string) int {
	diff := end.Sub(start)
	switch step {
	case "day":
		return int(diff.Hours()/24) + 1
	case "week":
		return int(diff.Hours()/(24*7)) + 1
	case "month":
		months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month())
		return months + 1
	}
	return int(diff.Hours()/24) + 1
}

// advanceDate moves a time.Time forward by one step unit.
func advanceDate(t time.Time, step string) time.Time {
	switch step {
	case "week":
		return t.AddDate(0, 0, 7)
	case "month":
		return t.AddDate(0, 1, 0)
	default: // "day"
		return t.AddDate(0, 0, 1)
	}
}

// formatDate formats a date according to the requested format.
func formatDate(t time.Time, format string) (string, error) {
	switch format {
	case "iso8601":
		return t.Format("2006-01-02"), nil
	case "unix":
		return strconv.FormatInt(t.Unix(), 10), nil
	case "human":
		return t.Format("Jan 2, 2006"), nil
	}
	return "", fmt.Errorf("unsupported format %q (valid: iso8601, unix, human)", format)
}
