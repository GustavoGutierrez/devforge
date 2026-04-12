# DevForge CLI/TUI

The `devforge` binary launches an interactive Bubble Tea interface for the stateless DevForge toolset.

## Main screens

- Generate/optimize images
- Generate favicon
- Video/audio processing
- UI to Markdown
- Markdown to PDF
- Text/data/crypto/HTTP/time/file/frontend/backend/code utilities (including ULID-aware ID generation, CIDR subnet calculation, color code conversion, color harmony palette generation, and CSS gradient generation)
- Settings and MCP setup

## Removed screens

The refactor removed DB-backed screens:

- Browse patterns
- Browse architectures
- Add record

## Config

The TUI reads the shared config file at `~/.config/devforge/config.json`.
