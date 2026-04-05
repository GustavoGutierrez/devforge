# DevForge MCP Server

`devforge-mcp` exposes DevForge tools over the MCP stdio transport.

## Notable tool groups

- Layout/design: `analyze_layout`, `suggest_layout`, `manage_tokens`, `suggest_color_palettes`
- Media/document: image/video/audio tools, `generate_favicon`, `generate_ui_image`, `ui2md`, `markdown_to_pdf`
- Utilities: text, data, crypto, HTTP, time, file, frontend, backend, code

## Runtime assumptions

- stdio only
- no database
- no Ollama or embedding services
- `dpf` must be available for media/document tools
