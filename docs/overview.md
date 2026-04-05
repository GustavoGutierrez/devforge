# DevForge Overview

DevForge is a stateless developer toolkit delivered as:

- an MCP stdio server (`devforge-mcp`)
- a Bubble Tea CLI/TUI (`devforge`)

## Main areas

- layout analysis and generation
- token planning
- color palette suggestions
- image/video/audio/document processing through `dpf`
- general-purpose developer utilities for text, data, crypto, HTTP, time, files, frontend, backend, and code

## Removed subsystem

The old persistence subsystem was intentionally removed:

- no SQLite/libSQL
- no FTS5 or vector search
- no Ollama/Nomic embeddings
- no stored patterns/architectures/audits
- no database browsing TUI screens

## Runtime model

DevForge now ships as a cleaner runtime bundle containing only the CLI, MCP server, and `dpf` binary.
