# AGENTS.md — DevForge MCP

## Overview

Go MCP server (`devforge-mcp`) + Bubble Tea CLI/TUI (`devforge`) providing 88 stateless developer
tools over stdio transport. Media tools delegate to **[dpf (DevPixelForge)](https://github.com/GustavoGutierrez/devpixelforge)**,
a Rust binary bundled in every release. No database, no embeddings, no persistent state.

## Build & Test

```bash
go build ./...               # compile all Go binaries
go test ./...                # run full test suite (CGO disabled)
bash scripts/install-dpf.sh  # update bin/dpf to latest devpixelforge release
```

Requires Go 1.24+. `bin/dpf` must exist locally for media tools to work.

## Project Structure

```
cmd/devforge/            CLI/TUI entrypoint (Bubble Tea)
cmd/devforge-mcp/        MCP server — register all tools here (main.go)
internal/tools/          Tool handlers (one file or subpackage per group)
internal/dpf/            dpf subprocess client (streaming protocol)
internal/version/        Version constant: const Current
scripts/                 Release, packaging, and maintenance scripts
.agents/skills/          Project-scoped agent skills
PRPs/                    Product Requirement Prompts — read before implementing features
VERSION                  Single source of truth for the release version
```

## Code Conventions

- MCP transport: **stdio only**. Do not add HTTP or WebSocket.
- Tool handlers return structured JSON errors — never panic.
- All tools are **stateless**: no DB writes, no side effects, no persistent state between calls.
- New tool: add handler in `internal/tools/`, register in `cmd/devforge-mcp/main.go`.
- Config: `~/.config/devforge/config.json` or `$DEV_FORGE_CONFIG`. Fields: `gemini_api_key`, `image_model`.
- CGO disabled. No cgo imports anywhere.
- Idiomatic Go: table-driven tests, errors as values, no global mutable state.
- Do not reintroduce DB-backed features (stored patterns, embeddings, audit logs) unless explicitly requested.

## Version Release

**Always use the bump script** — it updates all 4 locations atomically:

```bash
bash scripts/bump-version.sh X.Y.Z
```

| File | Field |
|------|-------|
| `VERSION` | plain-text version (source of truth for CI) |
| `internal/version/version.go` | `const Current` (TUI version display + update check) |
| `cmd/devforge-mcp/main.go` | `NewMCPServer` second argument |
| `README.md` | version badge |

`package_release_bundle.sh` fails at build time if any of these are out of sync.

Then publish:

```bash
go test ./...
git add VERSION internal/version/version.go cmd/devforge-mcp/main.go README.md
git commit -m "chore: bump to vX.Y.Z"
git tag vX.Y.Z
git push origin main && git push origin vX.Y.Z
```

CI builds Linux amd64 + macOS arm64 bundles, fetches the latest dpf automatically, and updates the
Homebrew tap. Use `workflow_dispatch` to republish an existing tag without re-tagging.

**Semantic versioning**: `PATCH` for fixes/docs, `MINOR` for new tools, `MAJOR` for breaking changes.

## New Tool Checklist

1. Add handler file or function to `internal/tools/` (or existing subpackage).
2. Register the tool in `cmd/devforge-mcp/main.go` → `registerTools()`.
3. Define the MCP tool schema (name, description, input schema) in the same handler file.
4. Add a table-driven unit test covering happy path and error cases.
5. Keep the handler stateless and side-effect-free.

## MCP Schema Rules (cross-model compatibility)

Every `mcp.WithArray(...)` call **must** include an `items` definition. Omitting it causes
`Invalid schema … array schema missing items` on OpenAI models (GPT-4o, GPT-5, Codex) and any
client that validates JSON Schema strictly.

| Array element type | Correct usage |
|--------------------|---------------|
| strings | `mcp.WithArray("x", ..., mcp.WithStringItems())` |
| numbers | `mcp.WithArray("x", ..., mcp.WithNumberItems())` |
| booleans | `mcp.WithArray("x", ..., mcp.WithBooleanItems())` |
| objects | `mcp.WithArray("x", ..., mcp.Items(map[string]any{"type": "object"}))` |

**Audit command** — run before every release to catch regressions:

```bash
# Reports WithArray calls where neither the same line nor the next line contain an items definition.
# Must return no output (zero violations) before releasing.
awk '
/WithArray/ {
  line = NR; content = $0
  getline nextline
  combined = content "\n" nextline
  if (combined !~ /WithStringItems|WithNumberItems|WithBooleanItems|WithStringEnumItems|mcp\.Items/) {
    print FILENAME ":" line ": " content
  }
}
' cmd/devforge-mcp/*.go
```

If the command returns any lines, add the appropriate `items` option before releasing.

## Workflow

- **SDD pipeline**: explore → propose → spec → design → tasks → apply → verify → archive.
  Load the matching skill with the `skill` tool at each phase.
- **Conventional commits**: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`.
- Read the relevant PRP in `PRPs/` before starting any feature implementation.
- Use the `publish-release` skill for the full release checklist.

## Available Skills

| Skill | Purpose |
|-------|---------|
| `publish-release` | Full release publishing checklist |
| `write-prp` | Write a Product Requirement Prompt |
| `golang-pro` | Idiomatic Go patterns, concurrency, generics |
| `plantuml-architecture` | Component / deployment diagrams |
| `plantuml-sequence` | Request-response and API flow diagrams |
| `plantuml-state` | State machine and lifecycle diagrams |
| `plantuml-activity` | Process and workflow activity diagrams |
| `writing-markdown` | Write or edit Markdown docs |
| `skill-creator` | Create a new project-scoped skill |
| `sdd-explore` `sdd-propose` `sdd-spec` `sdd-design` `sdd-tasks` `sdd-apply` `sdd-verify` `sdd-archive` | SDD pipeline phases |

## Security

- Never commit `~/.config/devforge/config.json` — it contains API keys.
- Tool handlers must not log secrets from config or request payloads.
- `dpf` binary path is resolved via `dpf.ResolveBinaryPath()` — do not exec arbitrary paths from user input.
