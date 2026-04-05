# AGENTS.md — DevForge MCP

## Overview

DevForge is a utility-focused Go MCP server and Bubble Tea CLI/TUI. The product is now stateless: no SQLite/libSQL/FTS/vector storage, no Ollama embeddings, and no bundled `devforge.db`.

## Build and test

```bash
go build ./...
go test ./...
```

`dpf` is still required for media tools and should live at `bin/dpf` during local development or next to the installed binaries in packaged releases.

## Key conventions

- MCP transport is stdio only.
- Tool handlers must return structured JSON errors instead of panicking.
- Config lives at `~/.config/devforge/config.json` or `DEV_FORGE_CONFIG`.
- Current config fields are `gemini_api_key` and `image_model`.
- New MCP tools go in `internal/tools/` and must be registered in `cmd/devforge-mcp/`.
- Keep CLI/TUI coherent with the stateless product direction.

## Tool surface

Keep and extend stateless utilities, including:

- layout/design helpers: `analyze_layout`, `suggest_layout`, `manage_tokens`, `suggest_color_palettes`
- media and document processing via `dpf`
- text/data/crypto/http/time/file/frontend/backend/code utilities

Do not reintroduce DB-backed features such as stored patterns, architecture persistence, audits persistence, embeddings, or browse/add-record database UIs unless explicitly requested as a new architectural direction.

## Release packaging

Release workflow publishes:

- `devforge_<version>_linux_amd64.tar.gz`
- `devforge_<version>_darwin_arm64.tar.gz`
- `checksums.txt`

Homebrew formula installs only:

- `devforge`
- `devforge-mcp`
- `dpf`
