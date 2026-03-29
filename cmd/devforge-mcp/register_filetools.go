// register_filetools.go registers all file and archive MCP tools with the server.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/filetools"
)

// registerFileTools adds the file_checksum, file_archive, file_diff,
// file_line_endings, and file_hex_view tools to the MCP server.
func registerFileTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── file_checksum ────────────────────────────────────────────
	s.AddTool(mcp.NewTool("file_checksum",
		mcp.WithDescription("Calculate the MD5, SHA-256, or SHA-512 checksum of a file by streaming it — the file is never fully loaded into memory."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute or relative path to the file")),
		mcp.WithString("algorithm", mcp.Description("Hash algorithm: md5 | sha256 | sha512 (default sha256)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := filetools.ChecksumInput{
			Path:      mcp.ParseString(req, "path", ""),
			Algorithm: mcp.ParseString(req, "algorithm", "sha256"),
		}
		return mcp.NewToolResultText(filetools.Checksum(ctx, in)), nil
	})

	// ── file_archive ─────────────────────────────────────────────
	s.AddTool(mcp.NewTool("file_archive",
		mcp.WithDescription("Create or extract zip or tar.gz archives. For create: provide source and output. For extract: provide archive and dest. Supports glob exclusion patterns."),
		mcp.WithString("operation", mcp.Required(), mcp.Description("create | extract")),
		mcp.WithString("format", mcp.Description("Archive format: zip | tar.gz (default zip)")),
		mcp.WithString("source", mcp.Description("Source file or directory path (required for create)")),
		mcp.WithString("output", mcp.Description("Output archive path (required for create)")),
		mcp.WithString("archive", mcp.Description("Archive file path to extract (required for extract)")),
		mcp.WithString("dest", mcp.Description("Destination directory for extracted files (required for extract)")),
		mcp.WithArray("exclude", mcp.Description("Glob patterns for files to exclude (e.g. ['*.log', 'tmp/'])"), mcp.WithStringItems()),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		in := filetools.ArchiveInput{
			Operation: mcp.ParseString(req, "operation", ""),
			Format:    mcp.ParseString(req, "format", "zip"),
			Source:    mcp.ParseString(req, "source", ""),
			Output:    mcp.ParseString(req, "output", ""),
			Archive:   mcp.ParseString(req, "archive", ""),
			Dest:      mcp.ParseString(req, "dest", ""),
		}
		if exRaw, ok := args["exclude"]; ok {
			data, _ := json.Marshal(exRaw)
			json.Unmarshal(data, &in.Exclude)
		}
		return mcp.NewToolResultText(filetools.Archive(ctx, in)), nil
	})

	// ── file_diff ────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("file_diff",
		mcp.WithDescription("Generate a unified diff between two files or text strings. Returns the diff, addition count, and deletion count."),
		mcp.WithString("a", mcp.Required(), mcp.Description("First file path (file mode) or text content (text mode)")),
		mcp.WithString("b", mcp.Required(), mcp.Description("Second file path (file mode) or text content (text mode)")),
		mcp.WithString("mode", mcp.Description("Comparison mode: file | text (default file)")),
		mcp.WithNumber("context_lines", mcp.Description("Number of unchanged context lines around each hunk (default 3)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := filetools.DiffInput{
			A:            mcp.ParseString(req, "a", ""),
			B:            mcp.ParseString(req, "b", ""),
			Mode:         mcp.ParseString(req, "mode", "file"),
			ContextLines: mcp.ParseInt(req, "context_lines", 3),
		}
		return mcp.NewToolResultText(filetools.Diff(ctx, in)), nil
	})

	// ── file_line_endings ────────────────────────────────────────
	s.AddTool(mcp.NewTool("file_line_endings",
		mcp.WithDescription("Detect, normalize, or convert line endings (LF/CRLF) in a file or text string. Returns statistics for detect; converted text or file path for normalize/convert."),
		mcp.WithString("input", mcp.Required(), mcp.Description("File path (file mode) or raw text (text mode)")),
		mcp.WithString("mode", mcp.Description("Input mode: file | text (default file)")),
		mcp.WithString("operation", mcp.Description("Operation: normalize | detect | convert (default detect)")),
		mcp.WithString("target", mcp.Description("Target line ending for normalize/convert: lf | crlf (default lf)")),
		mcp.WithString("output", mcp.Description("Output file path for file-mode normalize/convert (default: overwrite input)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := filetools.LineEndingsInput{
			Input:     mcp.ParseString(req, "input", ""),
			Mode:      mcp.ParseString(req, "mode", "file"),
			Operation: mcp.ParseString(req, "operation", "detect"),
			Target:    mcp.ParseString(req, "target", "lf"),
			Output:    mcp.ParseString(req, "output", ""),
		}
		return mcp.NewToolResultText(filetools.LineEndings(ctx, in)), nil
	})

	// ── file_hex_view ────────────────────────────────────────────
	s.AddTool(mcp.NewTool("file_hex_view",
		mcp.WithDescription("Display binary file content or base64-encoded bytes as a formatted hex+ASCII dump table. Supports offset and length for partial views."),
		mcp.WithString("input", mcp.Required(), mcp.Description("File path (file mode) or base64-encoded bytes (base64 mode)")),
		mcp.WithString("mode", mcp.Description("Input source: file | base64 (default file)")),
		mcp.WithNumber("offset", mcp.Description("Byte offset to start reading from (default 0)")),
		mcp.WithNumber("length", mcp.Description("Number of bytes to display (default 256)")),
		mcp.WithNumber("width", mcp.Description("Bytes per row in the hex dump (default 16)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := filetools.HexViewInput{
			Input:  mcp.ParseString(req, "input", ""),
			Mode:   mcp.ParseString(req, "mode", "file"),
			Offset: mcp.ParseInt(req, "offset", 0),
			Length: mcp.ParseInt(req, "length", 256),
			Width:  mcp.ParseInt(req, "width", 16),
		}
		return mcp.NewToolResultText(filetools.HexView(ctx, in)), nil
	})
}
