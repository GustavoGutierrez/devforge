# PRP: Developer Utilities Toolkit

> Version: 1.0 · Date: 2026-03-29 · Owner: GustavoGutierrez · Status: Draft

---

## 1. System

Infrastructure and stack context for this task:

- **Project**: DevForge MCP — Go MCP server + CLI/TUI (Bubble Tea), module `devforge-mcp`
- **Transport**: MCP stdio only (`github.com/mark3labs/mcp-go`)
- **Server entry point**: `cmd/devforge-mcp/main.go` — all MCP tools registered via `registerTools()`
- **CLI/TUI entry point**: `cmd/devforge/` — Bubble Tea, `lipgloss` styling, no raw ANSI
- **Tool handler pattern**: one file per tool under `internal/tools/`, method on `*tools.Server`, registered in `main.go`
- **Existing tool directories**: `internal/tools/*.go` (flat, current tools only)
- **New tool directories**: each utility group lives in its own subdirectory under `internal/tools/` to avoid mixing with existing tools:
  ```
  internal/tools/
    textenc/     # Group 1 — Text & Encoding
    datafmt/     # Group 2 — Data Format
    crypto/      # Group 3 — Security & Cryptography
    httptools/   # Group 4 — HTTP & Networking
    datetime/    # Group 5 — Date & Time
    filetools/   # Group 6 — File & Archive
    frontend/    # Group 7 — Frontend Utilities
    backend/     # Group 8 — Backend Utilities
    codetools/   # Group 9 — Code Utilities
  ```
- **Shared server**: `tools.Server` struct (in `internal/tools/tools.go`) — all groups receive it; no new shared state required for stateless utilities
- **DB**: SQLite WAL, `database/sql`, no ORM. Utilities in this PRP are **stateless** — no DB writes required
- **Config**: `~/.config/devforge/config.json` via `internal/config`; no new config keys required for this PRP
- **Build constraint**: `CGO_ENABLED=1` (required by `go-sqlite3`)
- **Language constraint**: all identifiers, comments, MCP tool descriptions, and documentation in **English**
- **Go skill**: apply `.agents/skills/golang-pro` patterns throughout implementation

---

## 2. Domain

### Business Problem

Developers spend significant time on deterministic micro-transformations — escaping strings, hashing data, parsing timestamps, validating JSON, building HTTP requests — that are too small for an LLM but repetitive enough to automate. DevForge MCP already automates UI and media work; this PRP extends it to cover the daily toolbox of backend and frontend developers.

Each utility must be:
- **Deterministic**: same input always produces the same output
- **Stateless**: no DB writes, no external network calls (except Group 4 HTTP executor)
- **Safe**: structured JSON error responses, never panic
- **Dual-surface**: callable from both the MCP server (AI agents) and the CLI/TUI (human developers)

### Tool Groups and MCP Tool Names

#### Group 1 — Text & Encoding (`internal/tools/textenc/`)

| MCP Tool | Description |
|----------|-------------|
| `text_escape` | Escape/unescape strings for JSON, JS, HTML, SQL targets |
| `text_slug` | Convert arbitrary text to URL-safe slugs |
| `text_uuid` | Generate UUID v4, nanoid, or random token |
| `text_base64` | Encode/decode Base64 (standard and URL-safe variants) |
| `text_url_encode` | Percent-encode/decode URL parameters and paths |
| `text_normalize` | Normalize whitespace, line endings, Unicode (NFC/NFD), strip BOM |
| `text_case` | Convert between camelCase, snake_case, kebab-case, PascalCase, SCREAMING_SNAKE |

#### Group 2 — Data Format (`internal/tools/datafmt/`)

| MCP Tool | Description |
|----------|-------------|
| `data_json_format` | Validate and pretty-print JSON; returns structured error with line/column |
| `data_yaml_convert` | Convert JSON ↔ YAML without type loss |
| `data_csv_convert` | Convert CSV ↔ JSON with configurable separator, headers, and type inference |
| `data_jsonpath` | Extract sub-structure from JSON using a JSONPath expression |
| `data_schema_validate` | Validate a JSON payload against a JSON Schema |
| `data_diff` | Compute structural diff between two JSON or YAML documents |

#### Group 3 — Security & Cryptography (`internal/tools/crypto/`)

| MCP Tool | Description |
|----------|-------------|
| `crypto_hash` | Hash a string or file with SHA-256, SHA-512, MD5, SHA-1 |
| `crypto_hmac` | Compute HMAC-SHA-256/SHA-512 given key + message |
| `crypto_jwt` | Decode JWT header/payload, verify signature and expiry, generate test tokens |
| `crypto_password` | Hash passwords with bcrypt or argon2id at secure default parameters |
| `crypto_keygen` | Generate RSA, EC (P-256/P-384), or Ed25519 key pairs in PEM or JWK format |
| `crypto_random` | Generate cryptographically secure random tokens, bytes, or OTPs |
| `crypto_mask` | Scan text/logs and redact patterns that match secrets, API keys, or passwords |

#### Group 4 — HTTP & Networking (`internal/tools/httptools/`)

| MCP Tool | Description |
|----------|-------------|
| `http_request` | Build and execute an HTTP request; return status, headers, body |
| `http_curl_convert` | Convert a `curl` command to Go `net/http`, TypeScript `fetch`, or Python `requests` snippet |
| `http_webhook_replay` | Re-send a saved HTTP payload (headers + body) to a target URL |
| `http_signed_url` | Generate an HMAC-signed URL with expiry (path + params + secret + TTL) |
| `http_url_parse` | Parse a URL into its components and rebuild it safely |

#### Group 5 — Date & Time (`internal/tools/datetime/`)

| MCP Tool | Description |
|----------|-------------|
| `time_convert` | Convert between Unix epoch, ISO 8601, RFC 3339, and human-readable formats across timezones |
| `time_diff` | Calculate duration between two timestamps; add/subtract time periods |
| `time_cron` | Validate and describe a cron expression in plain English; list next N execution times |
| `time_date_range` | Generate a list of dates (by day, week, or month) between start and end |

#### Group 6 — File & Archive (`internal/tools/filetools/`)

| MCP Tool | Description |
|----------|-------------|
| `file_checksum` | Calculate MD5/SHA-256/SHA-512 checksum of a file |
| `file_archive` | Create or extract zip/tar.gz archives with exclusion patterns |
| `file_diff` | Generate unified diff between two text files or strings |
| `file_line_endings` | Normalize line endings (CRLF → LF) and file encoding (→ UTF-8) |
| `file_hex_view` | Display binary file content as a hex+ASCII table |

#### Group 7 — Frontend Utilities (`internal/tools/frontend/`)

| MCP Tool | Description |
|----------|-------------|
| `frontend_color` | Convert colors between HEX, RGB, HSL, HSLA; compute contrast ratio |
| `frontend_css_unit` | Convert between px, rem, em, %, vw/vh with configurable base |
| `frontend_breakpoint` | Identify the responsive breakpoint for a given viewport width and generate media queries |
| `frontend_regex` | Test a regex against input; return all matches and capture groups |
| `frontend_locale_format` | Format numbers, dates, and currency using IETF locale strings |
| `frontend_icu_format` | Evaluate ICU message format strings with plural/select rules |

#### Group 8 — Backend Utilities (`internal/tools/backend/`)

| MCP Tool | Description |
|----------|-------------|
| `backend_sql_format` | Format and lint a SQL query (keywords, indentation, basic anti-patterns) |
| `backend_conn_string` | Build or parse database connection strings for PostgreSQL, MySQL, MongoDB, Redis |
| `backend_log_parse` | Parse and filter structured logs (JSON/NDJSON/Apache/Nginx) by field and time range |
| `backend_env_inspect` | Validate a `.env` file against an expected schema; generate `.env.example` |
| `backend_mq_payload` | Build, serialize, and format message queue payloads for Kafka, RabbitMQ, or SQS |

#### Group 9 — Code Utilities (`internal/tools/codetools/`)

| MCP Tool | Description |
|----------|-------------|
| `code_format` | Format a code snippet (Go, TypeScript, JSON, HTML, CSS) using embedded formatters |
| `code_metrics` | Report LOC, cyclomatic complexity estimate, and function count for a snippet |
| `code_template` | Render a Mustache/Handlebars-like template with a provided JSON context |

### Non-Functional Requirements

- **NFR-01**: Each utility group is a separate Go package under `internal/tools/<group>/`.
- **NFR-02**: All function and type names, comments, and MCP tool descriptions must be in English.
- **NFR-03**: Tool errors return `{"error": "message"}` JSON; never panic.
- **NFR-04**: No external network calls except `http_request` and `http_webhook_replay`.
- **NFR-05**: No new DB tables or migrations required.
- **NFR-06**: Every group must include `_test.go` file(s) with table-driven tests covering happy path and error cases.
- **NFR-07**: Every group must have a corresponding documentation file under `docs/tools/<group>.md`.
- **NFR-08**: `CGO_ENABLED=1 go build ./...` and `CGO_ENABLED=1 go test ./...` must pass after every group is added.

---

## 3. Task

### Objective

Implement 9 utility groups (49 MCP tools total) as stateless Go packages under `internal/tools/<group>/`, register each group's tools in `cmd/devforge-mcp/main.go`, expose them in the CLI/TUI, add docs, and validate compilation and tests per group.

### Implementation Order (dependency-free, each group is independent)

Each group is an independent implementation unit. Suggested order prioritizes highest developer impact and lowest risk:

1. **textenc** — pure string operations, zero dependencies
2. **datafmt** — standard library + `gopkg.in/yaml.v3`
3. **crypto** — `golang.org/x/crypto` (bcrypt, argon2)
4. **datetime** — standard library + `github.com/robfig/cron/v3`
5. **filetools** — standard library only
6. **httptools** — standard library `net/http`
7. **frontend** — pure math, no external deps
8. **backend** — standard library + `github.com/auxten/postgresql-parser` (optional for SQL lint)
9. **codetools** — `text/template` (Go standard library)

### Files to Create per Group (pattern for all 9 groups)

```
internal/tools/<group>/
  <group>.go           # Tool handler implementations; methods on *tools.Server or standalone funcs
  <group>_test.go      # Table-driven tests; one sub-test per tool
docs/tools/
  <group>.md           # Tool reference documentation
```

### Files to Modify

| File | Change |
|------|--------|
| `cmd/devforge-mcp/main.go` | Add `register<Group>Tools(s, app)` calls inside `registerTools()` for each group |
| `internal/tools/tools.go` | Add handler method stubs on `*Server` for each group, OR keep them as package-level functions that accept `*Server` — choose the pattern that best matches the existing code |
| `go.mod` / `go.sum` | Add `gopkg.in/yaml.v3`, `golang.org/x/crypto`, `github.com/robfig/cron/v3` |

### Out of Scope

- No new database tables or migrations
- No changes to existing tools (`internal/tools/*.go` flat files)
- No new config keys in `config.json`
- No authentication / rate limiting
- No streaming tool responses
- `code_format` for Go/TypeScript: use `gofmt`-style via `go/format` for Go; for TypeScript, apply simple indent normalization only (no full prettier dependency)
- `backend_mq_payload`: serialization helper only, no actual broker connections

### User Stories

- **US-01**: As an AI coding agent, I want to call `text_base64` with a string and get its Base64 encoding so I can build auth headers without guessing the format.
- **US-02**: As a developer using the TUI, I want to run `crypto_hash` on a file to verify its integrity after download.
- **US-03**: As an AI agent, I want to call `data_json_format` with malformed JSON and receive a structured error with line/column so I can show the user exactly where the syntax error is.
- **US-04**: As a backend developer, I want to call `time_cron` with a cron expression and get a human-readable description plus the next 5 execution times.
- **US-05**: As a frontend developer, I want `frontend_color` to convert a HEX color and tell me the contrast ratio against white so I can check WCAG compliance.
- **US-06**: As an AI agent, I want `crypto_mask` to sanitize a log blob before I send it to an external API.

### Acceptance Criteria

- **AC-01**: Given a valid input to any tool, the tool returns a structured JSON response with the correct result.
- **AC-02**: Given an invalid input, the tool returns `{"error": "<descriptive message>"}` — no panic, no stack trace.
- **AC-03**: `CGO_ENABLED=1 go build ./...` passes with zero errors after each group is added.
- **AC-04**: `CGO_ENABLED=1 go test ./internal/tools/<group>/...` passes for every group with at least one happy-path and one error-path test per tool.
- **AC-05**: Each MCP tool registered in `main.go` has a description in English that describes inputs, outputs, and constraints.
- **AC-06**: `docs/tools/<group>.md` exists for each group and documents all tools in the group (parameters, return schema, example).
- **AC-07**: No existing tests break (`CGO_ENABLED=1 go test ./...` is green).
- **AC-08**: No existing tool files under `internal/tools/*.go` are modified.

---

## 4. Interaction

- **Tone**: technical-direct — code-first, comments only when a decision needs justification
- **Language**: English — all identifiers, comments, MCP descriptions, and documentation
- **Go conventions**: follow `.agents/skills/golang-pro` — idiomatic Go, table-driven tests, no `any` type assertions without explicit comment, `sync.RWMutex` for any shared state introduced
- **Per-group delivery**: implement, test, and validate compilation for one group at a time before moving to the next
- **If a third-party library introduces a significant build cost**: prefer standard library alternative and document the tradeoff
- **If critical info is missing** (e.g., ambiguous input schema for a tool): apply the most conservative interpretation and add a `// TODO` comment — do not invent behavior

---

## 5. Response

### Deliverables per Group

For each of the 9 groups, deliver:

1. `internal/tools/<group>/<group>.go` — handler implementations
2. `internal/tools/<group>/<group>_test.go` — table-driven tests
3. `docs/tools/<group>.md` — tool reference documentation
4. Additions to `cmd/devforge-mcp/main.go` — tool registrations
5. `go.mod` / `go.sum` updates if new dependencies were added

### Quality Gates (must all pass before marking a group complete)

- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test ./internal/tools/<group>/...` exits 0, all subtests pass
- [ ] `CGO_ENABLED=1 go test ./...` exits 0 (no regressions)
- [ ] MCP tool descriptions are in English and clearly describe parameters and return value
- [ ] `docs/tools/<group>.md` exists and covers all tools in the group
- [ ] No new `panic()` calls introduced
- [ ] No `//nolint` or build-tag hacks to suppress real errors

### Final Quality Gate (all 9 groups complete)

- [ ] All 49 MCP tools appear in `registerTools()` in `main.go`
- [ ] `CGO_ENABLED=1 go test ./...` is green
- [ ] `CGO_ENABLED=1 go build ./...` is green
- [ ] `docs/tools/` has one `.md` file per group (9 total)

### Documentation to Update

- `docs/tools/<group>.md` — created per group (9 new files)
- `AGENTS.md` — update MCP Tools Reference table with all 49 new tools
- `README.md` — add Developer Utilities section listing the 9 groups

### Out of Scope for Response

- No UI/TUI implementation (CLI wiring is deferred to a follow-up PRP)
- No new database migrations
- No changes to existing tool files
- No Homebrew release bump (version bump and release are handled separately)
