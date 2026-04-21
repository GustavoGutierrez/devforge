// Package main — registration of data formatting MCP tools.
package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/datafmt"
)

// registerDataFmtTools registers all data-formatting tools with the MCP server.
func registerDataFmtTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── json_format ─────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("json_format",
			mcp.WithDescription("Pretty-print or re-indent a JSON string. Returns line/column info on syntax errors."),
			mcp.WithString("json", mcp.Required(), mcp.Description("Raw JSON string to format")),
			mcp.WithString("indent", mcp.Description("Indent string (default: two spaces)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(datafmt.FormatJSON(ctx, datafmt.FormatJSONInput{
				JSON:   mcp.ParseString(req, "json", ""),
				Indent: mcp.ParseString(req, "indent", "  "),
			})), nil
		},
	)

	// ── data_yaml_convert ────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("data_yaml_convert",
			mcp.WithDescription("Convert data between JSON and YAML formats."),
			mcp.WithString("input", mcp.Required(), mcp.Description("Input string to convert")),
			mcp.WithString("from", mcp.Required(), mcp.Description("Source format: json | yaml")),
			mcp.WithString("to", mcp.Required(), mcp.Description("Target format: json | yaml")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(datafmt.YAMLConvert(ctx, datafmt.YAMLConvertInput{
				Input: mcp.ParseString(req, "input", ""),
				From:  mcp.ParseString(req, "from", ""),
				To:    mcp.ParseString(req, "to", ""),
			})), nil
		},
	)

	// ── data_csv_convert ─────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("data_csv_convert",
			mcp.WithDescription("Convert between CSV and JSON formats. CSV→JSON produces an array of objects; JSON→CSV takes an array of objects."),
			mcp.WithString("input", mcp.Required(), mcp.Description("Input string to convert")),
			mcp.WithString("from", mcp.Required(), mcp.Description("Source format: csv | json")),
			mcp.WithString("to", mcp.Required(), mcp.Description("Target format: csv | json")),
			mcp.WithString("separator", mcp.Description("Field separator character (default: ,)")),
			mcp.WithBoolean("has_header", mcp.Description("CSV has a header row (default: true)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sep := mcp.ParseString(req, "separator", ",")
			hasHeader := mcp.ParseBoolean(req, "has_header", true)
			return mcp.NewToolResultText(datafmt.CSVConvert(ctx, datafmt.CSVConvertInput{
				Input:     mcp.ParseString(req, "input", ""),
				From:      mcp.ParseString(req, "from", ""),
				To:        mcp.ParseString(req, "to", ""),
				Separator: sep,
				HasHeader: hasHeader,
			})), nil
		},
	)

	// ── data_jsonpath ────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("data_jsonpath",
			mcp.WithDescription("Evaluate a JSONPath expression against a JSON document. Supports $, .field, [N], .*, [*]."),
			mcp.WithString("json", mcp.Required(), mcp.Description("JSON document to query")),
			mcp.WithString("path", mcp.Required(), mcp.Description("JSONPath expression (e.g. $.store.book[0].title)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(datafmt.JSONPath(ctx, datafmt.JSONPathInput{
				JSON: mcp.ParseString(req, "json", ""),
				Path: mcp.ParseString(req, "path", ""),
			})), nil
		},
	)

	// ── data_schema_validate ─────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("data_schema_validate",
			mcp.WithDescription("Validate a JSON document against a JSON Schema (supports type, required, properties, minimum/maximum, minLength/maxLength, enum, pattern, items, minItems/maxItems, additionalProperties)."),
			mcp.WithString("json", mcp.Required(), mcp.Description("JSON document to validate")),
			mcp.WithString("schema", mcp.Required(), mcp.Description("JSON Schema document")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(datafmt.SchemaValidate(ctx, datafmt.SchemaValidateInput{
				JSON:   mcp.ParseString(req, "json", ""),
				Schema: mcp.ParseString(req, "schema", ""),
			})), nil
		},
	)

	// ── data_diff ────────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("data_diff",
			mcp.WithDescription("Structural diff of two JSON or YAML objects at top-level keys. Returns added, removed, and changed entries."),
			mcp.WithString("a", mcp.Required(), mcp.Description("First document (JSON or YAML)")),
			mcp.WithString("b", mcp.Required(), mcp.Description("Second document (JSON or YAML)")),
			mcp.WithString("format", mcp.Description("Input format: json | yaml (default: json)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(datafmt.Diff(ctx, datafmt.DiffInput{
				A:      mcp.ParseString(req, "a", ""),
				B:      mcp.ParseString(req, "b", ""),
				Format: mcp.ParseString(req, "format", "json"),
			})), nil
		},
	)

	// ── fake_data (JSON Schema Faker) ──────────────────────────────────────
	s.AddTool(
		mcp.NewTool("fake_data",
			mcp.WithDescription("JSON Schema Faker — Generate fake data from a JSON Schema. Supports strings (email, ipv4, etc.), numbers, integers, booleans, objects, arrays, enums, and nested schemas. Returns 1-100 generated records."),
			mcp.WithString("schema", mcp.Required(), mcp.Description("JSON Schema definition to generate data from")),
			mcp.WithNumber("count", mcp.Description("Number of records to generate (1-100, default: 1)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(datafmt.FakeData(ctx, datafmt.FakeDataInput{
				Schema: mcp.ParseString(req, "schema", ""),
				Count:  mcp.ParseInt(req, "count", 1),
			})), nil
		},
	)
}
