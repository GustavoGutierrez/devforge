package datetime_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/datetime"
)

// ─── time_convert tests ──────────────────────────────────────────────────────

func TestTimeConvert_RFC3339ToUnix(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:    "2024-03-15T10:30:00Z",
		ToFormat: "unix",
	})
	var out datetime.TimeConvertOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Result == "" {
		t.Error("expected non-empty result")
	}
	if out.ToFormat != "unix" {
		t.Errorf("expected to_format=unix, got %q", out.ToFormat)
	}
	// 2024-03-15T10:30:00Z → Unix = 1710498600
	if out.Result != "1710498600" {
		t.Errorf("expected 1710498600, got %q", out.Result)
	}
}

func TestTimeConvert_UnixToRFC3339(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:      "1710498600",
		FromFormat: "unix",
		ToFormat:   "rfc3339",
		Timezone:   "UTC",
	})
	var out datetime.TimeConvertOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Result != "2024-03-15T10:30:00Z" {
		t.Errorf("expected 2024-03-15T10:30:00Z, got %q", out.Result)
	}
	if out.FromFormat != "unix" {
		t.Errorf("expected from_format=unix, got %q", out.FromFormat)
	}
}

func TestTimeConvert_UnixMsFormat(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:      "1710498600000",
		FromFormat: "unix_ms",
		ToFormat:   "rfc3339",
		Timezone:   "UTC",
	})
	var out datetime.TimeConvertOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Result != "2024-03-15T10:30:00Z" {
		t.Errorf("expected 2024-03-15T10:30:00Z, got %q", out.Result)
	}
}

func TestTimeConvert_HumanFormat(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:    "2024-03-15T10:30:00Z",
		ToFormat: "human",
		Timezone: "UTC",
	})
	var out datetime.TimeConvertOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if !strings.Contains(out.Result, "2024") {
		t.Errorf("expected year 2024 in human output, got %q", out.Result)
	}
}

func TestTimeConvert_AutoDetectsUnixMs(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:    "1710498600000",
		ToFormat: "rfc3339",
		Timezone: "UTC",
	})
	var out datetime.TimeConvertOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.FromFormat != "unix_ms" {
		t.Errorf("expected auto-detect unix_ms, got %q", out.FromFormat)
	}
}

func TestTimeConvert_MissingInput_ReturnsError(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input: "",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

func TestTimeConvert_InvalidTimezone_ReturnsError(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:    "2024-03-15T10:30:00Z",
		Timezone: "Not/A/Zone",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

func TestTimeConvert_UnparsableInput_ReturnsError(t *testing.T) {
	result := datetime.TimeConvert(context.Background(), datetime.TimeConvertInput{
		Input:    "not-a-date",
		ToFormat: "rfc3339",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

// ─── time_diff tests ─────────────────────────────────────────────────────────

func TestTimeDiff_BasicDiff(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start: "2024-01-01T00:00:00Z",
		End:   "2024-01-03T12:00:00Z",
	})
	var out datetime.TimeDiffResult
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// 2 days + 12 hours = 2.5 days = 60 hours
	if out.Days != 2.5 {
		t.Errorf("expected days=2.5, got %v", out.Days)
	}
	if out.Hours != 60 {
		t.Errorf("expected hours=60, got %v", out.Hours)
	}
	if out.Human == "" {
		t.Error("expected non-empty human description")
	}
}

func TestTimeDiff_AddDuration(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start:     "2024-01-01T00:00:00Z",
		Operation: "add",
		Duration:  "2h30m",
	})
	var out datetime.TimeAddResult
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Result != "2024-01-01T02:30:00Z" {
		t.Errorf("expected 2024-01-01T02:30:00Z, got %q", out.Result)
	}
}

func TestTimeDiff_SubtractDuration(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start:     "2024-01-10T00:00:00Z",
		Operation: "subtract",
		Duration:  "3 days",
	})
	var out datetime.TimeAddResult
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Result != "2024-01-07T00:00:00Z" {
		t.Errorf("expected 2024-01-07T00:00:00Z, got %q", out.Result)
	}
}

func TestTimeDiff_NegativeDiff(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start: "2024-01-05T00:00:00Z",
		End:   "2024-01-01T00:00:00Z",
	})
	var out datetime.TimeDiffResult
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Days != -4 {
		t.Errorf("expected days=-4, got %v", out.Days)
	}
}

func TestTimeDiff_MissingStart_ReturnsError(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start: "",
		End:   "2024-01-03T00:00:00Z",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

func TestTimeDiff_MissingEndForDiff_ReturnsError(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start:     "2024-01-01T00:00:00Z",
		Operation: "diff",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

func TestTimeDiff_AddMissingDuration_ReturnsError(t *testing.T) {
	result := datetime.TimeDiff(context.Background(), datetime.TimeDiffInput{
		Start:     "2024-01-01T00:00:00Z",
		Operation: "add",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

// ─── time_cron tests ─────────────────────────────────────────────────────────

func TestTimeCron_Describe_Daily(t *testing.T) {
	result := datetime.TimeCron(context.Background(), datetime.TimeCronInput{
		Expression: "0 9 * * *",
		Operation:  "describe",
	})
	var out datetime.TimeCronDescribeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if !out.Valid {
		t.Errorf("expected valid=true, got error: %s", out.Error)
	}
	if out.Description == "" {
		t.Error("expected non-empty description")
	}
}

func TestTimeCron_Describe_EveryMinute(t *testing.T) {
	result := datetime.TimeCron(context.Background(), datetime.TimeCronInput{
		Expression: "* * * * *",
		Operation:  "describe",
	})
	var out datetime.TimeCronDescribeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if !out.Valid {
		t.Errorf("expected valid=true, error: %s", out.Error)
	}
	if !strings.Contains(strings.ToLower(out.Description), "minute") {
		t.Errorf("expected 'minute' in description, got %q", out.Description)
	}
}

func TestTimeCron_Next_ReturnsCorrectCount(t *testing.T) {
	result := datetime.TimeCron(context.Background(), datetime.TimeCronInput{
		Expression: "0 9 * * *",
		Operation:  "next",
		Count:      3,
		From:       "2024-01-01T00:00:00Z",
	})
	var out datetime.TimeCronNextOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Count != 3 {
		t.Errorf("expected count=3, got %d", out.Count)
	}
	if len(out.Next) != 3 {
		t.Errorf("expected 3 entries, got %d", len(out.Next))
	}
	// Each should be RFC3339
	for _, ts := range out.Next {
		if !strings.Contains(ts, "T") {
			t.Errorf("expected RFC3339 format, got %q", ts)
		}
	}
}

func TestTimeCron_Next_At9AM(t *testing.T) {
	result := datetime.TimeCron(context.Background(), datetime.TimeCronInput{
		Expression: "0 9 * * *",
		Operation:  "next",
		Count:      1,
		From:       "2024-01-01T00:00:00Z",
	})
	var out datetime.TimeCronNextOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if len(out.Next) == 0 {
		t.Fatal("expected at least one next time")
	}
	// 2024-01-01T09:00:00Z
	if out.Next[0] != "2024-01-01T09:00:00Z" {
		t.Errorf("expected 2024-01-01T09:00:00Z, got %q", out.Next[0])
	}
}

func TestTimeCron_Describe_InvalidExpression(t *testing.T) {
	result := datetime.TimeCron(context.Background(), datetime.TimeCronInput{
		Expression: "not a cron",
		Operation:  "describe",
	})
	var out datetime.TimeCronDescribeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Valid {
		t.Error("expected valid=false for invalid expression")
	}
	if out.Error == "" {
		t.Error("expected non-empty error for invalid expression")
	}
}

func TestTimeCron_MissingExpression_ReturnsError(t *testing.T) {
	result := datetime.TimeCron(context.Background(), datetime.TimeCronInput{
		Expression: "",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}

// ─── time_date_range tests ───────────────────────────────────────────────────

func TestTimeDateRange_DailyStep(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start: "2024-01-01",
		End:   "2024-01-05",
		Step:  "day",
	})
	var out datetime.TimeDateRangeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Count != 5 {
		t.Errorf("expected 5 dates, got %d", out.Count)
	}
	if out.Dates[0] != "2024-01-01" {
		t.Errorf("expected first date 2024-01-01, got %q", out.Dates[0])
	}
	if out.Dates[4] != "2024-01-05" {
		t.Errorf("expected last date 2024-01-05, got %q", out.Dates[4])
	}
}

func TestTimeDateRange_WeeklyStep(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start: "2024-01-01",
		End:   "2024-01-29",
		Step:  "week",
	})
	var out datetime.TimeDateRangeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// 4 weeks + start = 5 dates: Jan 1, Jan 8, Jan 15, Jan 22, Jan 29
	if out.Count != 5 {
		t.Errorf("expected 5 weekly dates, got %d", out.Count)
	}
}

func TestTimeDateRange_MonthlyStep(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start: "2024-01-01",
		End:   "2024-03-01",
		Step:  "month",
	})
	var out datetime.TimeDateRangeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Count != 3 {
		t.Errorf("expected 3 monthly dates, got %d", out.Count)
	}
}

func TestTimeDateRange_UnixFormat(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start:  "2024-01-01",
		End:    "2024-01-02",
		Format: "unix",
	})
	var out datetime.TimeDateRangeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Count != 2 {
		t.Errorf("expected 2 dates, got %d", out.Count)
	}
	// Both should be numeric unix timestamps
	for _, d := range out.Dates {
		for _, c := range d {
			if c < '0' || c > '9' {
				t.Errorf("expected unix timestamp (digits only), got %q", d)
				break
			}
		}
	}
}

func TestTimeDateRange_HumanFormat(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start:  "2024-01-01",
		End:    "2024-01-01",
		Format: "human",
	})
	var out datetime.TimeDateRangeOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.Count != 1 {
		t.Errorf("expected 1 date, got %d", out.Count)
	}
	if !strings.Contains(out.Dates[0], "2024") {
		t.Errorf("expected year in human date, got %q", out.Dates[0])
	}
}

func TestTimeDateRange_ExceedsLimit_ReturnsError(t *testing.T) {
	// 2000 day range with daily step > 1000 limit
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start: "2020-01-01",
		End:   "2025-06-01",
		Step:  "day",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key for exceeding 1000 date limit")
	}
}

func TestTimeDateRange_EndBeforeStart_ReturnsError(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start: "2024-01-10",
		End:   "2024-01-01",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error for end before start")
	}
}

func TestTimeDateRange_MissingStart_ReturnsError(t *testing.T) {
	result := datetime.TimeDateRange(context.Background(), datetime.TimeDateRangeInput{
		Start: "",
		End:   "2024-01-05",
	})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error for missing start")
	}
}
