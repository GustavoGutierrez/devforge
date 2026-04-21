// Package datetime implements MCP tools for date and time operations.
// Tools: time_convert, time_diff, time_cron, time_date_range, current_date, current_week, week_number, calendar.
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

// ─── current_date ─────────────────────────────────────────────────────────

// CurrentDateInput is the input schema for the current_date tool.
type CurrentDateInput struct {
	Locale string `json:"locale"`
}

// CurrentDateOutput is the output schema for the current_date tool.
type CurrentDateOutput struct {
	Date              string `json:"date"`
	DayOfWeek         string `json:"day_of_week"`
	DayOfWeekShort    string `json:"day_of_week_short"`
	DayNumber         int    `json:"day_number"`
	Month             string `json:"month"`
	MonthShort        string `json:"month_short"`
	Year              int    `json:"year"`
	WeekOfYear        int    `json:"week_of_year"`
	WeekOfMonth       int    `json:"week_of_month"`
	IsWeekend         bool   `json:"is_weekend"`
	ISO8601           string `json:"iso8601"`
	UnixTimestamp     int64  `json:"unix_timestamp"`
}

// dayNamesEN are English weekday names (Sunday=0).
var dayNamesEN = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// dayNamesES are Spanish weekday names (Sunday=0).
var dayNamesES = []string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

// dayNamesESShort are Spanish abbreviated weekday names.
var dayNamesESShort = []string{"Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb"}

// monthNamesEN are English month names (January=1).
var monthNamesEN = []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}

// monthNamesES are Spanish month names.
var monthNamesES = []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}

// monthNamesESShort are Spanish abbreviated month names.
var monthNamesESShort = []string{"", "Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"}

// getDayNames returns day names based on locale.
func getDayNames(locale string) []string {
	if strings.ToLower(locale) == "es" {
		return dayNamesES
	}
	return dayNamesEN
}

// getDayNamesShort returns short day names based on locale.
func getDayNamesShort(locale string) []string {
	if strings.ToLower(locale) == "es" {
		return dayNamesESShort
	}
	return []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
}

// getMonthNames returns month names based on locale.
func getMonthNames(locale string) []string {
	if strings.ToLower(locale) == "es" {
		return monthNamesES
	}
	return monthNamesEN
}

// getMonthNamesShort returns short month names based on locale.
func getMonthNamesShort(locale string) []string {
	if strings.ToLower(locale) == "es" {
		return monthNamesESShort
	}
	return []string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
}

// weekOfYear returns the ISO week number (1-53) for a given date.
func weekOfYear(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

// weekOfMonth returns the week number within the month (1-5) for a given date.
func weekOfMonth(t time.Time) int {
	firstDay := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	firstWeekday := int(firstDay.Weekday())
	dayOfMonth := t.Day()

	if firstWeekday == 0 {
		firstWeekday = 7
	}
	weekNum := (dayOfMonth + firstWeekday - 2) / 7
	return weekNum + 1
}

// CurrentDate implements the current_date MCP tool.
// Returns the current date with full details in a human-readable format.
func CurrentDate(_ context.Context, input CurrentDateInput) string {
	locale := strings.ToLower(strings.TrimSpace(input.Locale))
	if locale == "" {
		locale = "en"
	}

	now := time.Now()
	dayIndex := int(now.Weekday())
	dayNames := getDayNames(locale)
	dayNamesShort := getDayNamesShort(locale)
	monthNames := getMonthNames(locale)
	monthNamesShort := getMonthNamesShort(locale)

	dayName := dayNames[dayIndex]
	dayNameShort := dayNamesShort[dayIndex]
	monthName := monthNames[now.Month()]
	monthNameShort := monthNamesShort[now.Month()]
	isWeekend := dayIndex == 0 || dayIndex == 6

	weekdayFormatted := dayName
	if locale == "es" {
		weekdayFormatted = dayName + " " + strconv.Itoa(now.Day()) + " de " + monthName + " de " + strconv.Itoa(now.Year())
	} else {
		weekdayFormatted = dayName + ", " + monthName + " " + strconv.Itoa(now.Day()) + ", " + strconv.Itoa(now.Year())
	}

	return resultJSON(CurrentDateOutput{
		Date:          weekdayFormatted,
		DayOfWeek:     dayName,
		DayOfWeekShort: dayNameShort,
		DayNumber:     now.Day(),
		Month:         monthName,
		MonthShort:    monthNameShort,
		Year:          now.Year(),
		WeekOfYear:    weekOfYear(now),
		WeekOfMonth:   weekOfMonth(now),
		IsWeekend:     isWeekend,
		ISO8601:       now.Format("2006-01-02"),
		UnixTimestamp: now.Unix(),
	})
}

// ─── current_week ─────────────────────────────────────────────────────────

// CurrentWeekInput is the input schema for the current_week tool.
type CurrentWeekInput struct {
	Locale    string `json:"locale"`
	Year      int    `json:"year"`
	WeekOfYear int   `json:"week_of_year"`
}

// CurrentWeekDay represents a single day in the week output.
type CurrentWeekDay struct {
	Date         string `json:"date"`
	DayName      string `json:"day_name"`
	DayNameShort string `json:"day_name_short"`
	DayNumber    int    `json:"day_number"`
	Month        string `json:"month"`
	MonthShort   string `json:"month_short"`
	Year         int    `json:"year"`
	IsWeekend   bool   `json:"is_weekend"`
	IsToday     bool   `json:"is_today"`
}

// CurrentWeekOutput is the output schema for the current_week tool.
type CurrentWeekOutput struct {
	WeekNumber    int            `json:"week_number"`
	WeekOfYear    int            `json:"week_of_year"`
	StartDate     string         `json:"start_date"`
	EndDate       string         `json:"end_date"`
	Locale        string         `json:"locale"`
	Days          []CurrentWeekDay `json:"days"`
	WorkingDays   []string       `json:"working_days"`
	WeekendDays   []string       `json:"weekend_days"`
}

// CurrentWeek implements the current_week MCP tool.
// Returns the days of the current week (or specified week) with working days highlighted.
func CurrentWeek(_ context.Context, input CurrentWeekInput) string {
	locale := strings.ToLower(strings.TrimSpace(input.Locale))
	if locale == "" {
		locale = "en"
	}

	now := time.Now()
	year := input.Year
	weekOfYearParam := input.WeekOfYear

	if year == 0 {
		year = now.Year()
	}
	if weekOfYearParam == 0 {
		_, weekOfYearParam = now.ISOWeek()
	}

	startOfWeek := getWeekStartDate(year, weekOfYearParam, locale)
	endOfWeek := startOfWeek.AddDate(0, 0, 6)

	dayNames := getDayNames(locale)
	dayNamesShort := getDayNamesShort(locale)
	monthNames := getMonthNames(locale)
	monthNamesShort := getMonthNamesShort(locale)

	days := make([]CurrentWeekDay, 7)
	workingDays := make([]string, 0, 5)
	weekendDays := make([]string, 0, 2)

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekdayFormatted := ""

	for i := 0; i < 7; i++ {
		currentDay := startOfWeek.AddDate(0, 0, i)
		dayIndex := int(currentDay.Weekday())
		isWeekend := dayIndex == 0 || dayIndex == 6
		isToday := currentDay.Equal(today)

		dayName := dayNames[dayIndex]
		dayNameShort := dayNamesShort[dayIndex]
		monthName := monthNames[currentDay.Month()]
		monthNameShort := monthNamesShort[currentDay.Month()]

		dayDateStr := dayName + " " + strconv.Itoa(currentDay.Day()) + " " + monthNameShort
		if locale == "es" {
			dayDateStr = dayName + " " + strconv.Itoa(currentDay.Day()) + " " + monthNameShort
		}

		if isToday {
			weekdayFormatted = dayName + " " + strconv.Itoa(currentDay.Day()) + " de " + monthName + " de " + strconv.Itoa(currentDay.Year())
			if locale == "en" {
				weekdayFormatted = dayName + ", " + monthName + " " + strconv.Itoa(currentDay.Day()) + ", " + strconv.Itoa(currentDay.Year())
			}
		}

		days[i] = CurrentWeekDay{
			Date:         currentDay.Format("2006-01-02"),
			DayName:      dayName,
			DayNameShort: dayNameShort,
			DayNumber:    currentDay.Day(),
			Month:        monthName,
			MonthShort:   monthNameShort,
			Year:         currentDay.Year(),
			IsWeekend:   isWeekend,
			IsToday:     isToday,
		}

		if isWeekend {
			weekendDays = append(weekendDays, dayDateStr)
		} else {
			workingDays = append(workingDays, dayDateStr)
		}
	}

	result := CurrentWeekOutput{
		WeekNumber:   weekOfMonth(startOfWeek.AddDate(0, 0, 3)),
		WeekOfYear:   weekOfYearParam,
		StartDate:    startOfWeek.Format("2006-01-02"),
		EndDate:      endOfWeek.Format("2006-01-02"),
		Locale:       locale,
		Days:         days,
		WorkingDays:  workingDays,
		WeekendDays:  weekendDays,
	}

	if locale == "es" {
		result.WeekNumber = weekOfMonth(startOfWeek.AddDate(0, 0, 3))
	}

	_ = weekdayFormatted

	return resultJSON(result)
}

// getWeekStartDate returns the first day (Monday) of the given ISO week.
func getWeekStartDate(year, week int, locale string) time.Time {
	thursday := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	isoWeek := week

	isoWeekMonday := thursday.AddDate(0, 0, (isoWeek-1)*7)
	daysFromThursday := (int(isoWeekMonday.Weekday()) - 4) % 7
	if daysFromThursday < 0 {
		daysFromThursday += 7
	}
	weekStart := isoWeekMonday.AddDate(0, 0, -daysFromThursday)

	if locale == "es" {
		weekStart = thursday.AddDate(0, 0, (isoWeek-1)*7)
		daysFromThursday = (int(weekStart.Weekday()) - 1) % 7
		if daysFromThursday < 0 {
			daysFromThursday += 7
		}
		weekStart = weekStart.AddDate(0, 0, -daysFromThursday)
	}

	return weekStart
}

// ─── week_number ──────────────────────────────────────────────────────────

// WeekNumberInput is the input schema for the week_number tool.
type WeekNumberInput struct {
	Date   string `json:"date"`
	Year   int    `json:"year"`
	Month  int    `json:"month"`
	Day    int    `json:"day"`
	Scope  string `json:"scope"`
}

// WeekNumberOutput is the output schema for the week_number tool.
type WeekNumberOutput struct {
	WeekOfYear     int    `json:"week_of_year"`
	WeekOfMonth    int    `json:"week_of_month"`
	Year           int    `json:"year"`
	Month          int    `json:"month"`
	Day            int    `json:"day"`
	Scope          string `json:"scope"`
	ISOWeekString  string `json:"iso_week_string"`
}

// WeekNumber implements the week_number MCP tool.
// Returns the week number for a given date or the current date.
func WeekNumber(_ context.Context, input WeekNumberInput) string {
	scope := strings.ToLower(strings.TrimSpace(input.Scope))
	if scope == "" {
		scope = "year"
	}

	var t time.Time
	var err error

	if strings.TrimSpace(input.Date) != "" {
		t, _, err = parseTimestamp(input.Date, "auto", time.UTC)
		if err != nil {
			return errResult("could not parse date: " + err.Error())
		}
	} else if input.Year != 0 && input.Month != 0 && input.Day != 0 {
		if input.Month < 1 || input.Month > 12 {
			return errResult("month must be between 1 and 12")
		}
		if input.Day < 1 || input.Day > 31 {
			return errResult("day must be between 1 and 31")
		}
		t = time.Date(input.Year, time.Month(input.Month), input.Day, 0, 0, 0, 0, time.UTC)
	} else {
		t = time.Now()
	}

	year, week := t.ISOWeek()
	weekNumOfMonth := weekOfMonth(t)

	isoWeekString := fmt.Sprintf("%d-W%02d", year, week)

	return resultJSON(WeekNumberOutput{
		WeekOfYear:    week,
		WeekOfMonth:   weekNumOfMonth,
		Year:          t.Year(),
		Month:         int(t.Month()),
		Day:           t.Day(),
		Scope:         scope,
		ISOWeekString: isoWeekString,
	})
}

// ─── calendar ─────────────────────────────────────────────────────────────

// CalendarInput is the input schema for the calendar tool.
type CalendarInput struct {
	Year         int    `json:"year"`
	Month        int    `json:"month"`
	Locale       string `json:"locale"`
	StartOfWeek  string `json:"start_of_week"`
}

// CalendarDay represents a single day in the calendar output.
type CalendarDay struct {
	Date         string `json:"date"`
	DayNumber    int    `json:"day_number"`
	DayName      string `json:"day_name"`
	DayNameShort string `json:"day_name_short"`
	Month        int    `json:"month"`
	Year         int    `json:"year"`
	IsWeekend   bool   `json:"is_weekend"`
	IsToday     bool   `json:"is_today"`
	IsCurrentMonth bool `json:"is_current_month"`
	WeekNumber   int    `json:"week_number"`
}

// CalendarWeek represents a week row in the calendar.
type CalendarWeek struct {
	WeekNumber int          `json:"week_number"`
	Days       []CalendarDay `json:"days"`
}

// CalendarOutput is the output schema for the calendar tool.
type CalendarOutput struct {
	Year         int            `json:"year"`
	Month        int            `json:"month"`
	MonthName    string         `json:"month_name"`
	Locale       string         `json:"locale"`
	StartOfWeek  string         `json:"start_of_week"`
	TotalDays    int            `json:"total_days"`
	FirstDayOfWeek int          `json:"first_day_of_week"`
	Weeks        []CalendarWeek `json:"weeks"`
	WorkingDays  []string       `json:"working_days"`
	WeekendDays  []string       `json:"weekend_days"`
}

// Calendar implements the calendar MCP tool.
// Returns a monthly calendar with days organized by week.
func Calendar(_ context.Context, input CalendarInput) string {
	locale := strings.ToLower(strings.TrimSpace(input.Locale))
	if locale == "" {
		locale = "en"
	}

	startOfWeek := strings.ToLower(strings.TrimSpace(input.StartOfWeek))
	if startOfWeek == "" {
		startOfWeek = "monday"
	}

	now := time.Now()
	year := input.Year
	month := input.Month

	if year == 0 {
		year = now.Year()
	}
	if month == 0 {
		month = int(now.Month())
	}

	if month < 1 || month > 12 {
		return errResult("month must be between 1 and 12")
	}
	if year < 1 || year > 9999 {
		return errResult("year must be between 1 and 9999")
	}

	monthNames := getMonthNames(locale)
	monthName := monthNames[time.Month(month)]

	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	totalDays := lastDay.Day()

	dayNames := getDayNames(locale)
	dayNamesShort := getDayNamesShort(locale)

	firstWeekday := int(firstDay.Weekday())
	if startOfWeek == "monday" {
		if firstWeekday == 0 {
			firstWeekday = 7
		}
		firstWeekday--
	}

	weeks := []CalendarWeek{}
	workingDays := make([]string, 0, 23)
	weekendDays := make([]string, 0, 10)

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	weekNum, _ := firstDay.ISOWeek()
	currentWeek := 0
	dayCount := 0
	emptyDaysBefore := firstWeekday
	if emptyDaysBefore < 0 {
		emptyDaysBefore = 0
	}

	weeks = append(weeks, CalendarWeek{
		WeekNumber: weekNum,
		Days:       []CalendarDay{},
	})

	for dayCount < totalDays+emptyDaysBefore {
		if dayCount > 0 && (dayCount-emptyDaysBefore)%7 == 0 {
			currentWeek++
			weekNum++
			weeks = append(weeks, CalendarWeek{
				WeekNumber: weekNum,
				Days:       []CalendarDay{},
			})
		}

		dayInMonth := dayCount - emptyDaysBefore + 1
		if dayInMonth < 1 || dayInMonth > totalDays {
			weeks[currentWeek].Days = append(weeks[currentWeek].Days, CalendarDay{
				Date:          "",
				DayNumber:     0,
				DayName:       "",
				DayNameShort:  "",
				Month:         month,
				Year:          year,
				IsWeekend:     false,
				IsToday:       false,
				IsCurrentMonth: false,
				WeekNumber:    weekNum,
			})
		} else {
			currentDate := time.Date(year, time.Month(month), dayInMonth, 0, 0, 0, 0, time.UTC)
			dayIndex := int(currentDate.Weekday())
			isWeekend := dayIndex == 0 || dayIndex == 6
			isToday := currentDate.Equal(today)

			dayDateStr := dayNames[dayIndex] + " " + strconv.Itoa(dayInMonth)

			calendarDay := CalendarDay{
				Date:           currentDate.Format("2006-01-02"),
				DayNumber:      dayInMonth,
				DayName:        dayNames[dayIndex],
				DayNameShort:   dayNamesShort[dayIndex],
				Month:          month,
				Year:           year,
				IsWeekend:      isWeekend,
				IsToday:        isToday,
				IsCurrentMonth: true,
				WeekNumber:     weekNum,
			}
			weeks[currentWeek].Days = append(weeks[currentWeek].Days, calendarDay)

			if isWeekend {
				weekendDays = append(weekendDays, dayDateStr)
			} else {
				workingDays = append(workingDays, dayDateStr)
			}
		}
		dayCount++
	}

	for len(weeks) > 0 && len(weeks[len(weeks)-1].Days) < 7 {
		emptyDay := CalendarDay{
			Date:          "",
			DayNumber:     0,
			DayName:       "",
			DayNameShort:  "",
			Month:         month,
			Year:          year,
			IsWeekend:     false,
			IsToday:       false,
			IsCurrentMonth: false,
			WeekNumber:    weeks[len(weeks)-1].WeekNumber,
		}
		weeks[len(weeks)-1].Days = append(weeks[len(weeks)-1].Days, emptyDay)
	}

	return resultJSON(CalendarOutput{
		Year:            year,
		Month:           month,
		MonthName:       monthName,
		Locale:          locale,
		StartOfWeek:    startOfWeek,
		TotalDays:       totalDays,
		FirstDayOfWeek: firstWeekday,
		Weeks:           weeks,
		WorkingDays:     workingDays,
		WeekendDays:     weekendDays,
	})
}
