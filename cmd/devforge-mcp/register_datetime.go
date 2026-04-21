// register_datetime.go registers all datetime MCP tools with the server.
package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/datetime"
)

func registerDateTimeTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── time_convert ─────────────────────────────────────────────
	s.AddTool(mcp.NewTool("time_convert",
		mcp.WithDescription("Convert a timestamp between formats (unix, unix_ms, iso8601, rfc3339, human). Auto-detects input format when from_format is omitted."),
		mcp.WithString("input", mcp.Required(), mcp.Description("Timestamp to convert")),
		mcp.WithString("from_format", mcp.Description("Source format: unix | unix_ms | iso8601 | rfc3339 | human | auto (default: auto)")),
		mcp.WithString("to_format", mcp.Description("Target format: unix | unix_ms | iso8601 | rfc3339 | human (default: rfc3339)")),
		mcp.WithString("timezone", mcp.Description("IANA timezone name for output (default: UTC)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.TimeConvertInput{
			Input:      mcp.ParseString(req, "input", ""),
			FromFormat: mcp.ParseString(req, "from_format", "auto"),
			ToFormat:   mcp.ParseString(req, "to_format", "rfc3339"),
			Timezone:   mcp.ParseString(req, "timezone", "UTC"),
		}
		return mcp.NewToolResultText(datetime.TimeConvert(ctx, input)), nil
	})

	// ── time_diff ────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("time_diff",
		mcp.WithDescription("Calculate the difference between two timestamps, or add/subtract a duration from a timestamp."),
		mcp.WithString("start", mcp.Required(), mcp.Description("Start timestamp (ISO8601 or RFC3339)")),
		mcp.WithString("end", mcp.Description("End timestamp for diff operation (ISO8601 or RFC3339)")),
		mcp.WithString("operation", mcp.Description("Operation: diff | add | subtract (default: diff)")),
		mcp.WithString("duration", mcp.Description("Duration for add/subtract: Go format (2h30m) or English (3 days)")),
		mcp.WithString("unit", mcp.Description("Preferred unit for output: auto | seconds | minutes | hours | days | weeks (default: auto)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.TimeDiffInput{
			Start:     mcp.ParseString(req, "start", ""),
			End:       mcp.ParseString(req, "end", ""),
			Operation: mcp.ParseString(req, "operation", "diff"),
			Duration:  mcp.ParseString(req, "duration", ""),
			Unit:      mcp.ParseString(req, "unit", "auto"),
		}
		return mcp.NewToolResultText(datetime.TimeDiff(ctx, input)), nil
	})

	// ── time_cron ────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("time_cron",
		mcp.WithDescription("Describe a cron expression in plain English, or compute the next N execution times."),
		mcp.WithString("expression", mcp.Required(), mcp.Description("Cron expression (5 or 6 fields)")),
		mcp.WithString("operation", mcp.Description("Operation: describe | next (default: describe)")),
		mcp.WithNumber("count", mcp.Description("Number of next times to compute (default: 5)")),
		mcp.WithString("from", mcp.Description("Reference datetime for next computation (default: now)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.TimeCronInput{
			Expression: mcp.ParseString(req, "expression", ""),
			Operation:  mcp.ParseString(req, "operation", "describe"),
			Count:      mcp.ParseInt(req, "count", 5),
			From:       mcp.ParseString(req, "from", ""),
		}
		return mcp.NewToolResultText(datetime.TimeCron(ctx, input)), nil
	})

	// ── time_date_range ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("time_date_range",
		mcp.WithDescription("Generate a list of dates between two ISO8601 dates with a configurable step (day, week, month). Capped at 1000 dates."),
		mcp.WithString("start", mcp.Required(), mcp.Description("Start date (ISO8601 format: YYYY-MM-DD)")),
		mcp.WithString("end", mcp.Required(), mcp.Description("End date (ISO8601 format: YYYY-MM-DD)")),
		mcp.WithString("step", mcp.Description("Step size: day | week | month (default: day)")),
		mcp.WithString("format", mcp.Description("Output format: iso8601 | unix | human (default: iso8601)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.TimeDateRangeInput{
			Start:  mcp.ParseString(req, "start", ""),
			End:    mcp.ParseString(req, "end", ""),
			Step:   mcp.ParseString(req, "step", "day"),
			Format: mcp.ParseString(req, "format", "iso8601"),
		}
		return mcp.NewToolResultText(datetime.TimeDateRange(ctx, input)), nil
	})

	// ── current_date ─────────────────────────────────────────────
	s.AddTool(mcp.NewTool("current_date",
		mcp.WithDescription("Get the current date with full details. Returns day of week, day number, month, year, week of year/month, and whether it's a weekend. Useful for agents to know the exact current date."),
		mcp.WithString("locale", mcp.Description("Language for output: en (English, default) | es (Spanish)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.CurrentDateInput{
			Locale: mcp.ParseString(req, "locale", "en"),
		}
		return mcp.NewToolResultText(datetime.CurrentDate(ctx, input)), nil
	})

	// ── current_week ─────────────────────────────────────────────
	s.AddTool(mcp.NewTool("current_week",
		mcp.WithDescription("Get the days of the current week (or a specified week) with working days and weekend highlighted. Returns full details for each day including date, day name, day number, month, and weekend/working day status."),
		mcp.WithString("locale", mcp.Description("Language for output: en (English, default) | es (Spanish)")),
		mcp.WithNumber("year", mcp.Description("Year (default: current year)")),
		mcp.WithNumber("week_of_year", mcp.Description("ISO week number 1-53 (default: current week)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.CurrentWeekInput{
			Locale:    mcp.ParseString(req, "locale", "en"),
			Year:      mcp.ParseInt(req, "year", 0),
			WeekOfYear: mcp.ParseInt(req, "week_of_year", 0),
		}
		return mcp.NewToolResultText(datetime.CurrentWeek(ctx, input)), nil
	})

	// ── week_number ──────────────────────────────────────────────
	s.AddTool(mcp.NewTool("week_number",
		mcp.WithDescription("Get the week number for a specific date, month, or the current date. Returns week of year (ISO 1-53) and week of month (1-5). Use when planning timelines or schedules to know which week a date falls on."),
		mcp.WithString("date", mcp.Description("Date in any parseable format (ISO8601, RFC3339, human). Takes precedence over year/month/day.")),
		mcp.WithNumber("year", mcp.Description("Year (used when date is not provided)")),
		mcp.WithNumber("month", mcp.Description("Month 1-12 (used when date is not provided)")),
		mcp.WithNumber("day", mcp.Description("Day 1-31 (used when date is not provided)")),
		mcp.WithString("scope", mcp.Description("Scope for output: year | month (default: year)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.WeekNumberInput{
			Date:  mcp.ParseString(req, "date", ""),
			Year:  mcp.ParseInt(req, "year", 0),
			Month: mcp.ParseInt(req, "month", 0),
			Day:   mcp.ParseInt(req, "day", 0),
			Scope: mcp.ParseString(req, "scope", "year"),
		}
		return mcp.NewToolResultText(datetime.WeekNumber(ctx, input)), nil
	})

	// ── calendar ─────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("calendar",
		mcp.WithDescription("Generate a monthly calendar with all days organized by week. Returns week numbers, day names, day numbers, weekend vs working day markers. Use for planning schedules, timelines, and project milestones."),
		mcp.WithNumber("year", mcp.Description("Year (default: current year)")),
		mcp.WithNumber("month", mcp.Description("Month 1-12 (default: current month)")),
		mcp.WithString("locale", mcp.Description("Language for output: en (English, default) | es (Spanish)")),
		mcp.WithString("start_of_week", mcp.Description("First day of week: monday (default) | sunday")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := datetime.CalendarInput{
			Year:        mcp.ParseInt(req, "year", 0),
			Month:       mcp.ParseInt(req, "month", 0),
			Locale:      mcp.ParseString(req, "locale", "en"),
			StartOfWeek: mcp.ParseString(req, "start_of_week", "monday"),
		}
		return mcp.NewToolResultText(datetime.Calendar(ctx, input)), nil
	})
}
