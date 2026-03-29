// register_httptools.go registers all HTTP and networking MCP tools.
package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"dev-forge-mcp/internal/tools/httptools"
)

func registerHTTPTools(s *mcpserver.MCPServer, _ *mcpApp) {
	// ── http_request ─────────────────────────────────────────────
	s.AddTool(mcp.NewTool("http_request",
		mcp.WithDescription("Perform an HTTP request and return status, headers, body, and duration."),
		mcp.WithString("url", mcp.Required(), mcp.Description("Target URL")),
		mcp.WithString("method", mcp.Description("HTTP method (default GET)")),
		mcp.WithObject("headers", mcp.Description("Request headers as key/value pairs")),
		mcp.WithString("body", mcp.Description("Request body")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Request timeout in seconds (default 30)")),
		mcp.WithBoolean("follow_redirects", mcp.Description("Follow HTTP redirects (default true)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := httptools.HTTPRequestInput{
			URL:             mcp.ParseString(req, "url", ""),
			Method:          mcp.ParseString(req, "method", "GET"),
			Body:            mcp.ParseString(req, "body", ""),
			TimeoutSeconds:  mcp.ParseInt(req, "timeout_seconds", 30),
			FollowRedirects: mcp.ParseBoolean(req, "follow_redirects", true),
		}
		if hdrs, ok := args["headers"].(map[string]interface{}); ok {
			input.Headers = make(map[string]string, len(hdrs))
			for k, v := range hdrs {
				if s, ok := v.(string); ok {
					input.Headers[k] = s
				}
			}
		}
		return mcp.NewToolResultText(httptools.HTTPRequest(ctx, input)), nil
	})

	// ── http_curl_convert ────────────────────────────────────────
	s.AddTool(mcp.NewTool("http_curl_convert",
		mcp.WithDescription("Convert a curl command to a code snippet in Go, TypeScript, or Python."),
		mcp.WithString("curl", mcp.Required(), mcp.Description("curl command string to convert")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target language: go | typescript | python")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := httptools.HTTPCurlConvertInput{
			Curl:   mcp.ParseString(req, "curl", ""),
			Target: mcp.ParseString(req, "target", ""),
		}
		return mcp.NewToolResultText(httptools.HTTPCurlConvert(ctx, input)), nil
	})

	// ── http_webhook_replay ──────────────────────────────────────
	s.AddTool(mcp.NewTool("http_webhook_replay",
		mcp.WithDescription("Replay a saved webhook payload to a target URL."),
		mcp.WithString("url", mcp.Required(), mcp.Description("Target URL")),
		mcp.WithString("method", mcp.Description("HTTP method (default POST)")),
		mcp.WithObject("headers", mcp.Description("Request headers as key/value pairs")),
		mcp.WithString("body", mcp.Description("Webhook payload body")),
		mcp.WithNumber("timeout_seconds", mcp.Description("Request timeout in seconds (default 30)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := httptools.HTTPWebhookReplayInput{
			URL:            mcp.ParseString(req, "url", ""),
			Method:         mcp.ParseString(req, "method", "POST"),
			Body:           mcp.ParseString(req, "body", ""),
			TimeoutSeconds: mcp.ParseInt(req, "timeout_seconds", 30),
		}
		if hdrs, ok := args["headers"].(map[string]interface{}); ok {
			input.Headers = make(map[string]string, len(hdrs))
			for k, v := range hdrs {
				if s, ok := v.(string); ok {
					input.Headers[k] = s
				}
			}
		}
		return mcp.NewToolResultText(httptools.HTTPWebhookReplay(ctx, input)), nil
	})

	// ── http_signed_url ──────────────────────────────────────────
	s.AddTool(mcp.NewTool("http_signed_url",
		mcp.WithDescription("Generate a signed URL or HMAC-SHA256 signature for secure access control."),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL to sign")),
		mcp.WithString("secret", mcp.Required(), mcp.Description("HMAC secret key")),
		mcp.WithNumber("expiry_seconds", mcp.Description("Seconds until expiry (default 3600)")),
		mcp.WithString("method", mcp.Description("Delivery method: query | header (default query)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		input := httptools.HTTPSignedURLInput{
			URL:           mcp.ParseString(req, "url", ""),
			Secret:        mcp.ParseString(req, "secret", ""),
			ExpirySeconds: mcp.ParseInt(req, "expiry_seconds", 3600),
			Method:        mcp.ParseString(req, "method", "query"),
		}
		return mcp.NewToolResultText(httptools.HTTPSignedURL(ctx, input)), nil
	})

	// ── http_url_parse ───────────────────────────────────────────
	s.AddTool(mcp.NewTool("http_url_parse",
		mcp.WithDescription("Parse a URL into its components or build a URL from components."),
		mcp.WithString("url", mcp.Description("URL to parse (required for parse action)")),
		mcp.WithString("action", mcp.Description("Action: parse | build (default parse)")),
		mcp.WithObject("components", mcp.Description("URL components for build action: scheme, host, path, query (object), fragment")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := argsMap(req)
		input := httptools.HTTPURLParseInput{
			URL:    mcp.ParseString(req, "url", ""),
			Action: mcp.ParseString(req, "action", "parse"),
		}
		if comps, ok := args["components"].(map[string]interface{}); ok {
			// Deserialize via JSON round-trip to get the proper nested type
			b, _ := json.Marshal(comps)
			var parsed map[string]interface{}
			if err := json.Unmarshal(b, &parsed); err == nil {
				input.Components = parsed
			}
		}
		return mcp.NewToolResultText(httptools.HTTPURLParse(ctx, input)), nil
	})
}
