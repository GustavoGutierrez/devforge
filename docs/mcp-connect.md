# Connecting DevForge to MCP clients

Run `devforge-mcp` as a stdio MCP server.

Minimal generic client entry:

```json
{
  "mcpServers": {
    "devforge": {
      "command": "/path/to/devforge-mcp"
    }
  }
}
```

## Example tool calls

### `analyze_layout`

```json
{
  "name": "analyze_layout",
  "arguments": {
    "markup": "<main><img src=\"hero.png\" alt=\"Hero\"></main>",
    "page_type": "landing",
    "device_focus": "both",
    "stack": {
      "css_mode": "plain-css",
      "framework": "vanilla"
    }
  }
}
```

### `manage_tokens`

```json
{
  "name": "manage_tokens",
  "arguments": {
    "mode": "plan-update",
    "css_mode": "tailwind-v4",
    "scope": "colors",
    "proposal": {
      "--color-primary": "#2563eb"
    }
  }
}
```

### `http_request`

```json
{
  "name": "http_request",
  "arguments": {
    "method": "GET",
    "url": "https://example.com",
    "headers": {},
    "body": ""
  }
}
```
