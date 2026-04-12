# DevForge MCP Server

`devforge-mcp` exposes DevForge tools over the MCP stdio transport.

## Notable tool groups

- Media/document: image/video/audio tools, `generate_favicon`, `generate_ui_image`, `ui2md`, `markdown_to_pdf`
- Color utilities: `color_code_convert`, `color_harmony_palette`, `css_gradient_generate`
- Utilities: text, data, crypto, HTTP, time, file, frontend, backend, code

## Runtime assumptions

- stdio only
- no database
- no Ollama or embedding services
- `dpf` must be available for media/document tools
