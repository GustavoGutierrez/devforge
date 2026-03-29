// Package main — frontend tool registration for the DevForge MCP server.
// This file registers all Group 7 (Frontend Utilities) MCP tools.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/frontend"
)

// registerFrontendTools registers all frontend utility tools with the MCP server.
func registerFrontendTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── frontend_color ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("frontend_color",
		mcp.WithDescription("Convert colors between HEX, RGB, HSL, HSLA, RGBA formats and compute WCAG 2.1 contrast ratio. Returns the converted color and optionally contrast_ratio, wcag_aa, wcag_aaa when 'against' color is provided."),
		mcp.WithString("color", mcp.Required(), mcp.Description("Source color: #RRGGBB, #RGB, rgb(r,g,b), or hsl(h,s%,l%)")),
		mcp.WithString("to", mcp.Description("Target format: hex | rgb | hsl | hsla | rgba (default: hex)")),
		mcp.WithNumber("alpha", mcp.Description("Alpha channel 0.0-1.0 for rgba/hsla output (default: 1.0)")),
		mcp.WithString("against", mcp.Description("Optional second color for WCAG contrast ratio computation")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := frontendArgsMap(req)
		in := frontend.ColorInput{
			Color:   mcp.ParseString(req, "color", ""),
			To:      mcp.ParseString(req, "to", "hex"),
			Alpha:   frontendNumVal(args, "alpha", 1.0),
			Against: mcp.ParseString(req, "against", ""),
		}
		return mcp.NewToolResultText(frontend.Color(ctx, in)), nil
	})

	// ── frontend_css_unit ───────────────────────────────────────
	s.AddTool(mcp.NewTool("frontend_css_unit",
		mcp.WithDescription("Convert CSS values between units: px, rem, em, percent, vw, vh. Returns the numeric result, both unit labels, and a formatted string like '1rem'. Provide base_font_size, viewport_width, viewport_height, and parent_size for accurate em/percent/vw/vh conversions."),
		mcp.WithNumber("value", mcp.Required(), mcp.Description("Source numeric value")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Source unit: px | rem | em | percent | vw | vh")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Target unit: px | rem | em | percent | vw | vh")),
		mcp.WithNumber("base_font_size", mcp.Description("Root font size in px for rem conversions (default: 16)")),
		mcp.WithNumber("viewport_width", mcp.Description("Viewport width in px for vw conversions (default: 1440)")),
		mcp.WithNumber("viewport_height", mcp.Description("Viewport height in px for vh conversions (default: 900)")),
		mcp.WithNumber("parent_size", mcp.Description("Parent element size in px for em/percent conversions (default: 16)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := frontendArgsMap(req)
		in := frontend.CSSUnitInput{
			Value:          frontendNumVal(args, "value", 0),
			From:           mcp.ParseString(req, "from", ""),
			To:             mcp.ParseString(req, "to", ""),
			BaseFontSize:   frontendNumVal(args, "base_font_size", 16),
			ViewportWidth:  frontendNumVal(args, "viewport_width", 1440),
			ViewportHeight: frontendNumVal(args, "viewport_height", 900),
			ParentSize:     frontendNumVal(args, "parent_size", 16),
		}
		return mcp.NewToolResultText(frontend.CSSUnit(ctx, in)), nil
	})

	// ── frontend_breakpoint ─────────────────────────────────────
	s.AddTool(mcp.NewTool("frontend_breakpoint",
		mcp.WithDescription("Identify the responsive breakpoint for a viewport width and optionally generate the corresponding CSS media query. Supports Tailwind v4 (sm=640, md=768, lg=1024, xl=1280, 2xl=1536), Bootstrap 5 (xs=0, sm=576, md=768, lg=992, xl=1200, xxl=1400), or a custom breakpoint set."),
		mcp.WithNumber("width", mcp.Required(), mcp.Description("Viewport width in pixels")),
		mcp.WithString("system", mcp.Description("Breakpoint system: tailwind | bootstrap | custom (default: tailwind)")),
		mcp.WithObject("custom_breakpoints", mcp.Description("Custom breakpoints as {name: minWidthPx} pairs (required when system=custom)")),
		mcp.WithBoolean("generate_query", mcp.Description("Include generated @media query in response (default: true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := frontendArgsMap(req)
		in := frontend.BreakpointInput{
			Width:         int(frontendNumVal(args, "width", 0)),
			System:        mcp.ParseString(req, "system", "tailwind"),
			GenerateQuery: mcp.ParseBoolean(req, "generate_query", true),
		}
		if bpMap, ok := args["custom_breakpoints"].(map[string]any); ok {
			in.CustomBreakpoints = make(map[string]int, len(bpMap))
			for k, v := range bpMap {
				switch n := v.(type) {
				case float64:
					in.CustomBreakpoints[k] = int(n)
				}
			}
		}
		return mcp.NewToolResultText(frontend.Breakpoint(ctx, in)), nil
	})

	// ── frontend_regex ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("frontend_regex",
		mcp.WithDescription("Test, match, or replace using a regular expression. Flags: 'i' for case-insensitive, 'm' for multiline, 'g' for global (all matches). Operations: 'test' returns {matches, count}; 'match' returns array of {full, groups, index}; 'replace' returns {result, count}."),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Regular expression pattern (without delimiters)")),
		mcp.WithString("input", mcp.Required(), mcp.Description("Input string to test against")),
		mcp.WithString("flags", mcp.Description("Flags: i (case-insensitive), m (multiline), g (global/all matches). Combine: 'ig'")),
		mcp.WithString("operation", mcp.Description("Operation: test | match | replace (default: test)")),
		mcp.WithString("replacement", mcp.Description("Replacement string for replace operation")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := frontend.RegexInput{
			Pattern:     mcp.ParseString(req, "pattern", ""),
			Input:       mcp.ParseString(req, "input", ""),
			Flags:       mcp.ParseString(req, "flags", ""),
			Operation:   mcp.ParseString(req, "operation", "test"),
			Replacement: mcp.ParseString(req, "replacement", ""),
		}
		return mcp.NewToolResultText(frontend.Regex(ctx, in)), nil
	})

	// ── frontend_locale_format ──────────────────────────────────
	s.AddTool(mcp.NewTool("frontend_locale_format",
		mcp.WithDescription("Format numbers, dates, currencies, and percentages according to IETF locale conventions. Supported locales: en-US, en-GB, de-DE, fr-FR, es-ES, pt-BR, ja-JP, zh-CN. Returns {formatted, locale, kind}."),
		mcp.WithString("value", mcp.Required(), mcp.Description("Value to format: numeric string or ISO 8601 date/datetime string")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("Format kind: number | currency | date | time | datetime | percent")),
		mcp.WithString("locale", mcp.Description("IETF locale tag (default: en-US)")),
		mcp.WithString("currency", mcp.Description("ISO 4217 currency code (e.g. USD, EUR, GBP) — required for kind=currency")),
		mcp.WithObject("options", mcp.Description("Additional options: decimal_places (int), format (custom Go time layout string)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := frontendArgsMap(req)
		in := frontend.LocaleFormatInput{
			Value:    mcp.ParseString(req, "value", ""),
			Kind:     mcp.ParseString(req, "kind", ""),
			Locale:   mcp.ParseString(req, "locale", "en-US"),
			Currency: mcp.ParseString(req, "currency", ""),
		}
		if optsRaw, ok := args["options"].(map[string]any); ok {
			in.Options = optsRaw
		}
		return mcp.NewToolResultText(frontend.LocaleFormat(ctx, in)), nil
	})

	// ── frontend_icu_format ─────────────────────────────────────
	s.AddTool(mcp.NewTool("frontend_icu_format",
		mcp.WithDescription("Evaluate an ICU message format string with variable bindings. Supports: {variable} simple substitution, {variable, plural, one{# item} other{# items}} for pluralization, {variable, select, male{He} female{She} other{They}} for selection. Returns {result}."),
		mcp.WithString("template", mcp.Required(), mcp.Description("ICU message format template string")),
		mcp.WithObject("values", mcp.Required(), mcp.Description("Variable bindings as key-value pairs. Numeric values must be numbers for plural rules.")),
		mcp.WithString("locale", mcp.Description("Locale for plural rules (default: en)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := frontendArgsMap(req)
		in := frontend.ICUFormatInput{
			Template: mcp.ParseString(req, "template", ""),
			Locale:   mcp.ParseString(req, "locale", "en"),
		}
		if valuesRaw, ok := args["values"]; ok {
			data, _ := json.Marshal(valuesRaw)
			// Best-effort unmarshal — invalid JSON falls back to nil values map.
			_ = json.Unmarshal(data, &in.Values)
		}
		return mcp.NewToolResultText(frontend.ICUFormat(ctx, in)), nil
	})
}

// frontendArgsMap extracts the arguments map from a CallToolRequest.
// Defined here to avoid conflicts with argsMap in main.go (same package).
func frontendArgsMap(req mcp.CallToolRequest) map[string]any {
	if m, ok := req.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// frontendNumVal extracts a float64 from a map with a fallback.
func frontendNumVal(m map[string]any, key string, fallback float64) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case float32:
			return float64(n)
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return fallback
}
