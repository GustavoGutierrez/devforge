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

### `code_json_to_types`

```json
{
  "name": "code_json_to_types",
  "arguments": {
    "json": "{\"id\":1,\"name\":\"Ada\",\"active\":true}",
    "language": "typescript",
    "root_name": "User"
  }
}
```

### `code_ast_explorer`

```json
{
  "name": "code_ast_explorer",
  "arguments": {
    "code": "export class UserService {}\nconst handler = () => 1",
    "language": "typescript"
  }
}
```

### `frontend_svg_optimize`

```json
{
  "name": "frontend_svg_optimize",
  "arguments": {
    "svg": "<svg><!--comment--><metadata>meta</metadata><path d=\"M0 0\"/></svg>"
  }
}
```

### `frontend_image_base64`

```json
{
  "name": "frontend_image_base64",
  "arguments": {
    "path": "./assets/icon.png",
    "data_uri": true
  }
}
```

### `css_gradient_generate`

```json
{
  "name": "css_gradient_generate",
  "arguments": {
    "gradient_type": "linear",
    "angle": 90,
    "stops": [
      { "color": "#2a7b9b", "position": 0 },
      { "color": "#57c785", "position": 50 },
      { "color": "#eddd53", "position": 100 }
    ]
  }
}
```

### `color_code_convert`

```json
{
  "name": "color_code_convert",
  "arguments": {
    "color": "#3B82F6",
    "from": "hex",
    "to": "oklch"
  }
}
```

### `color_harmony_palette`

```json
{
  "name": "color_harmony_palette",
  "arguments": {
    "base_color": "#FF6B6B",
    "harmony": "analogous",
    "spread": 30
  }
}
```

### `generate_ui_image`

```json
{
  "name": "generate_ui_image",
  "arguments": {
    "prompt": "Modern SaaS dashboard with side nav and KPI cards",
    "style": "mockup",
    "width": 1280,
    "height": 720,
    "output_path": "/tmp/dashboard.png"
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

### `backend_cidr_subnet`

```json
{
  "name": "backend_cidr_subnet",
  "arguments": {
    "cidr": "10.0.0.0/24",
    "include_all": true,
    "limit": 16
  }
}
```
