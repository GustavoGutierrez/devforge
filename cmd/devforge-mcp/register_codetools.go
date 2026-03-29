// register_codetools.go registers the Code Utilities MCP tools with the server.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/codetools"
)

// registerCodeTools registers all code_* MCP tools onto the given server.
// These tools are stateless and require no app dependencies.
func registerCodeTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── code_format ──────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("code_format",
			mcp.WithDescription("Format source code for a given language. Supports Go (using go/format), JSON (configurable indent), TypeScript/JavaScript (indent normalization), HTML (heuristic tag indentation), and CSS (one property per line)."),
			mcp.WithString("code", mcp.Required(), mcp.Description("Source code to format")),
			mcp.WithString("language", mcp.Required(), mcp.Description("Language: go | typescript | json | html | css")),
			mcp.WithNumber("indent_size", mcp.Description("Indent size in spaces (default: 2; ignored for Go)")),
			mcp.WithBoolean("use_tabs", mcp.Description("Use tabs for indentation (default: false; Go always uses tabs)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := argsMap(req)
			input := codetools.FormatInput{
				Code:     mcp.ParseString(req, "code", ""),
				Language: mcp.ParseString(req, "language", ""),
				UseTabs:  mcp.ParseBoolean(req, "use_tabs", false),
			}
			if is, ok := args["indent_size"].(float64); ok {
				input.IndentSize = int(is)
			} else {
				input.IndentSize = 2
			}
			return mcp.NewToolResultText(codetools.Format(ctx, input)), nil
		},
	)

	// ── code_metrics ─────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("code_metrics",
			mcp.WithDescription("Compute code quality metrics: lines of code (LOC), source lines (SLOC), blank lines, comment lines, function count, and cyclomatic complexity estimate. Uses go/ast for Go; regex heuristics for other languages."),
			mcp.WithString("code", mcp.Required(), mcp.Description("Source code to analyze")),
			mcp.WithString("language", mcp.Required(), mcp.Description("Language: go | typescript | python | generic")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(codetools.Metrics(ctx, codetools.MetricsInput{
				Code:     mcp.ParseString(req, "code", ""),
				Language: mcp.ParseString(req, "language", ""),
			})), nil
		},
	)

	// ── code_template ────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("code_template",
			mcp.WithDescription("Render a template with JSON context bindings. Supports Go text/template syntax (engine: go) and a minimal Mustache interpreter (engine: mustache) with {{variable}}, {{#section}}...{{/section}}, {{^inverted}}...{{/inverted}}, and {{! comment }} tags."),
			mcp.WithString("template", mcp.Required(), mcp.Description("Template string to render")),
			mcp.WithString("context", mcp.Required(), mcp.Description("JSON object with variable bindings")),
			mcp.WithString("engine", mcp.Description("Template engine: go | mustache (default: go)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := argsMap(req)
			// Support passing context as an object (auto-serialize to JSON string).
			ctxStr := mcp.ParseString(req, "context", "{}")
			if ctxStr == "" {
				if ctxObj, ok := args["context"]; ok {
					if b, err := json.Marshal(ctxObj); err == nil {
						ctxStr = string(b)
					}
				}
			}
			return mcp.NewToolResultText(codetools.Template(ctx, codetools.TemplateInput{
				Template: mcp.ParseString(req, "template", ""),
				Context:  ctxStr,
				Engine:   mcp.ParseString(req, "engine", "go"),
			})), nil
		},
	)
}
