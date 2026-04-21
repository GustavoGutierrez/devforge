// register_textenc.go registers the Text & Encoding MCP tools with the server.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/textenc"
)

// registerTextEncTools registers all text_* MCP tools onto the given server.
// None of these tools require app state — they are pure stateless functions.
func registerTextEncTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── text_escape ──────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_escape",
			mcp.WithDescription("Escape or unescape a string for JSON, JavaScript, HTML, or SQL targets."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input string to escape or unescape")),
			mcp.WithString("target", mcp.Description("Escaping target: json | js | html | sql (default: json)")),
			mcp.WithString("operation", mcp.Description("Operation: escape | unescape (default: escape)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.Escape(ctx, textenc.EscapeInput{
				Text:      mcp.ParseString(req, "text", ""),
				Target:    mcp.ParseString(req, "target", "json"),
				Operation: mcp.ParseString(req, "operation", "escape"),
			})), nil
		},
	)

	// ── text_slug ────────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_slug",
			mcp.WithDescription("Convert arbitrary text into a URL-safe slug by normalizing Unicode, stripping non-alphanumeric characters, and collapsing separators."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input text to slugify")),
			mcp.WithString("separator", mcp.Description("Word separator character (default: \"-\")")),
			mcp.WithBoolean("lower", mcp.Description("Convert to lowercase (default: true)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.Slug(ctx, textenc.SlugInput{
				Text:      mcp.ParseString(req, "text", ""),
				Separator: mcp.ParseString(req, "separator", "-"),
				Lower:     mcp.ParseBoolean(req, "lower", true),
			})), nil
		},
	)

	// ── text_uuid ────────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_uuid",
			mcp.WithDescription("Generate one or more unique identifiers: UUID v4, ULID, nanoid (URL-safe random string), or a hex-encoded random token."),
			mcp.WithString("kind", mcp.Description("Kind of identifier to generate: uuid4 | ulid | nanoid | token (default: uuid4)")),
			mcp.WithNumber("length", mcp.Description("Length of the generated value for nanoid and token (default: 21)")),
			mcp.WithNumber("count", mcp.Description("Number of identifiers to generate (default: 1, max: 1000)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.UUID(ctx, textenc.UUIDInput{
				Kind:   mcp.ParseString(req, "kind", "uuid4"),
				Length: mcp.ParseInt(req, "length", 21),
				Count:  mcp.ParseInt(req, "count", 1),
			})), nil
		},
	)

	// ── text_base64 ──────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_base64",
			mcp.WithDescription("Encode or decode a string using Base64 (standard RFC 4648 or URL-safe variant)."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input string to encode or decode")),
			mcp.WithString("variant", mcp.Description("Base64 variant: standard | urlsafe (default: standard)")),
			mcp.WithString("operation", mcp.Description("Operation: encode | decode (default: encode)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.Base64(ctx, textenc.Base64Input{
				Text:      mcp.ParseString(req, "text", ""),
				Variant:   mcp.ParseString(req, "variant", "standard"),
				Operation: mcp.ParseString(req, "operation", "encode"),
			})), nil
		},
	)

	// ── text_url_encode ──────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_url_encode",
			mcp.WithDescription("Percent-encode or decode a URL query parameter or path segment."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input string to encode or decode")),
			mcp.WithString("operation", mcp.Description("Operation: encode | decode (default: encode)")),
			mcp.WithString("mode", mcp.Description("Encoding mode: query (uses + for spaces) | path (uses %20 for spaces) (default: query)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.URLEncode(ctx, textenc.URLEncodeInput{
				Text:      mcp.ParseString(req, "text", ""),
				Operation: mcp.ParseString(req, "operation", "encode"),
				Mode:      mcp.ParseString(req, "mode", "query"),
			})), nil
		},
	)

	// ── text_normalize ───────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_normalize",
			mcp.WithDescription("Apply one or more normalization operations to text: trim whitespace, normalize line endings, strip UTF-8 BOM, or apply Unicode normalization forms (NFC, NFD, NFKC, NFKD)."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input text to normalize")),
			mcp.WithArray("operations", mcp.Required(), mcp.Description("Ordered list of operations to apply: trim_whitespace | normalize_newlines | strip_bom | nfc | nfd | nfkc | nfkd"), mcp.WithStringItems()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := argsMap(req)
			input := textenc.NormalizeInput{
				Text: mcp.ParseString(req, "text", ""),
			}
			if opsRaw, ok := args["operations"]; ok {
				data, _ := json.Marshal(opsRaw)
				_ = json.Unmarshal(data, &input.Operations)
			}
			return mcp.NewToolResultText(textenc.Normalize(ctx, input)), nil
		},
	)

	// ── text_case ────────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_case",
			mcp.WithDescription("Convert text between naming conventions: camelCase, snake_case, kebab-case, PascalCase, or SCREAMING_SNAKE_CASE. Handles spaces, hyphens, underscores, and camelCase boundaries as word separators."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input text to convert")),
			mcp.WithString("target_case", mcp.Required(), mcp.Description("Target naming convention: camel | snake | kebab | pascal | screaming_snake")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.Case(ctx, textenc.CaseInput{
				Text:       mcp.ParseString(req, "text", ""),
				TargetCase: mcp.ParseString(req, "target_case", ""),
			})), nil
		},
	)

	// ── text_stats ───────────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("text_stats",
			mcp.WithDescription("Count words, characters, sentences, and paragraphs in text. Returns word count, character count (with and without spaces), sentence count, paragraph count, unique words, most frequent word, and average word length."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Input text to analyze")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText(textenc.TextStats(ctx, textenc.TextStatsInput{
				Text: mcp.ParseString(req, "text", ""),
			})), nil
		},
	)
}
