# PRP: Tools Efficiency and Reuse — Cross-Cutting Optimization

> Version: 1.0 · Date: 2026-05-24 · Owner: GustavoGutierrez · Status: Done

---

## 1. System

Infrastructure and stack context for this task:

- **Project**: DevForge MCP — Go MCP server (`devforge-mcp`) + Bubble Tea CLI/TUI (`devforge`), Go 1.24+
- **Transport**: MCP stdio only (`github.com/mark3labs/mcp-go`). Do not introduce HTTP/WebSocket.
- **Server entry point**: `cmd/devforge-mcp/main.go` — `registerTools()` at line 118 fans out to 10 sub-functions in `register_*.go` plus ~27 inline registrations for image/video/audio/UI tools (lines 132–771).
- **Tool registration files**: `cmd/devforge-mcp/register_textenc.go`, `register_datafmt.go`, `register_cryptoutil.go`, `register_httptools.go`, `register_datetime.go`, `register_filetools.go`, `register_colors.go`, `register_frontend.go`, `register_backend.go`, `register_codetools.go`.
- **Handler packages**: `internal/tools/{textenc,datafmt,cryptoutil,httptools,datetime,filetools,colors,frontend,backend,codetools}` plus flat tool files (`image_tools.go`, `video_tools.go`, `audio_tools.go`, `optimize_images.go`, `generate_favicon.go`, `ui2md.go`) in `internal/tools/`.
- **Media subsystem**: `internal/dpf/` — Rust subprocess (`bin/dpf`). `StreamClient` (lines 524–605) keeps one long-lived process with `sync.Mutex` serializing every call. `Client.Execute()` (line 212) still spawns per-call processes for some image ops.
- **Config**: loaded once at startup via `internal/config.Load()` (`config.go:53`). Gemini key/model are guarded by `sync.RWMutex` on `mcpApp` and hot-reloaded via `configure_gemini`.
- **Server type**: `tools.Server` holds only `*dpf.StreamClient` plus delegated Gemini service.
- **Build constraint**: `CGO_ENABLED=0`. No CGO imports may be introduced.
- **Statelessness invariant**: no DB, no embeddings, no persistent state. This PRP MUST preserve this property.
- **Schema rule**: every `mcp.WithArray(...)` MUST include `items` (already enforced; audit currently returns zero violations — keep it that way).
- **Skills to apply**: `golang-pro`, `go-testing`, `writing-markdown`.

---

## 2. Domain

### Problem

DevForge MCP currently ships ~88 stateless tools across 13 handler packages. The codebase grew by accretion (PRP 002 added the bulk of utility groups in one shot), and three classes of friction have accumulated:

1. **Duplication**: the `errResult` / `resultJSON` helpers are copy-pasted verbatim into 13 packages. The root `internal/tools/tools.go` has near-equivalent `errorJSON` / `mustJSON` that no sub-package imports. The `cmd/devforge-mcp` package defines `argsMap` once and `frontendArgsMap` as a near-clone to avoid a naming conflict. The DPF nil-guard (`if s.DPF == nil { return errorJSON(...) }`) appears 24+ times across image/video/audio handlers.
2. **Per-call work that should be cached**: `internal/tools/codetools/codetools.go` recompiles fixed (non-user-driven) regexes inside `countPattern`, `countFunctionsRegex`, and `countComplexityRegex` (lines 619, 628, 631, 634) on every invocation. These patterns are language-detection heuristics and should be package-level vars.
3. **LLM tool-selection ambiguity**: image, video, and audio tools have one-sentence descriptions ("Crop an image to specific dimensions.", "Trim audio by timestamps.") with no disambiguation cues, no example inputs, and no contrast with adjacent tools. Text/datetime/backend tools have richer descriptions that materially help model routing. The descriptions are the LLM's only signal — uneven quality directly hurts tool-call accuracy.

Secondary problems:

- **DPF call serialization**: `StreamClient.Mutex` serializes every dpf call globally. Concurrent MCP requests for unrelated media ops queue behind each other.
- **Context cancellation ignored**: pure-Go handlers accept `ctx` and discard it (`_ context.Context`). Long-running ops (regex/template eval, fake_data, heavy text processing) cannot be cancelled by an upstream timeout.
- **Test gaps**: 6 top-level tool files have zero test coverage: `audio_tools.go`, `video_tools.go`, `image_tools.go`, `optimize_images.go`, `generate_favicon.go`, `ui2md.go`. Error paths for the DPF-nil guard have never been exercised by a test.

### Goals (in priority order)

- **G1 — Reusability**: a single source of truth for tool result/error envelopes, DPF readiness check, and parameter parsing. Adding tool #89 should not require copying boilerplate from a neighbor file.
- **G2 — Efficiency**: kill repeated compile work on hot paths. Make DPF call serialization scope-correct (or document why it's intentional).
- **G3 — Clarity for LLMs**: every tool description includes purpose, key params, one example use case, and a disambiguation hint when an adjacent tool exists.
- **G4 — Test coverage for DPF wrappers**: each media tool file has at least an error-path test for the DPF-nil guard and one happy-path test against a recorded fixture or fake `StreamClient`.

### Non-functional constraints

- **Backward compatibility**: zero changes to tool names, MCP schemas (input/output shapes), or CLI/TUI flags. Existing MCP clients keep working byte-for-byte on responses.
- **Stateless**: no new global mutable state, no DB, no on-disk caches.
- **CGO disabled**: solutions must not pull in CGO transitively.
- **Stdio-only transport**: no transport changes.
- **Performance ceiling**: this PRP should never regress p50 latency on any tool. Cached regexes and shared helpers must be measured (benchmarks) on the codetools path.

### Out of scope

- Adding new tools.
- Changing MCP transport.
- Reintroducing DB-backed features.
- Rewriting dpf protocol or refactoring the Rust binary.
- Splitting `cmd/devforge-mcp/main.go` into smaller files for purely organizational reasons.

---

## 3. Task

### Objective

Land a cross-cutting refactor that (a) introduces a small `internal/tools/toolsutil` shared helper package, (b) caches package-level regexes in `codetools`, (c) levels up image/video/audio tool descriptions to LLM-friendly quality, (d) wires real context cancellation in pure-Go handlers, (e) decides on and implements either a DPF call pool or a documented serialization contract, and (f) closes test gaps for media tool files.

### Workstreams

#### W1 — Shared helpers package `internal/tools/toolsutil`

- **Create**: `internal/tools/toolsutil/toolsutil.go`
- **Contents**:
  - `func ErrResult(msg string) string` — returns the canonical JSON error envelope used today.
  - `func ResultJSON(v any) string` — returns the canonical JSON-marshal envelope used today.
  - `func RequireDPF(client *dpf.StreamClient) (string, bool)` — returns `("", true)` when client is non-nil, or `(ErrResult("dpf binary not available"), false)` when nil. Caller uses `if msg, ok := toolsutil.RequireDPF(s.DPF); !ok { return msg }`.
- **Migrate**: replace the 13 in-package `errResult` / `resultJSON` definitions with imports of `toolsutil`. Delete dead duplicates. Leave the root `internal/tools/tools.go` `errorJSON`/`mustJSON` only if other internal-package callers still use them; otherwise inline-replace and delete.
- **Migrate DPF guard**: replace the 24+ ad-hoc nil checks in `image_tools.go`, `video_tools.go`, `audio_tools.go`, `optimize_images.go`, `generate_favicon.go` with `toolsutil.RequireDPF`.
- **Acceptance**: zero remaining definitions of `errResult` / `resultJSON` outside `toolsutil`. Zero remaining direct `if s.DPF == nil` checks in handler files.

#### W2 — Regex caching in `codetools`

- **File**: `internal/tools/codetools/codetools.go`
- **Change**: hoist all `regexp.MustCompile(...)` calls that compile fixed (non-user-supplied) patterns from inside `countPattern`, `countFunctionsRegex`, `countComplexityRegex` (lines 619, 628, 631, 634) to package-level `var (...)` block at the top of the file. Naming: `rePythonFunc`, `reJSFunc`, `reGoFunc`, `reComplexityJS`, etc. — descriptive.
- **Leave alone**: regexes derived from user input (`regex_test` in `frontend/frontend.go:631`, template parsing at `codetools.go:728`) — those are correctly per-call.
- **Add**: a Go benchmark `BenchmarkCodeMetrics` in `internal/tools/codetools/codetools_test.go` over a ~500-line fixture. Must show non-regression (target: ≥ 30% faster on small inputs where compile dominated).
- **Acceptance**: `go test -bench=. ./internal/tools/codetools/...` succeeds. Numbers recorded in the PRP completion comment.

#### W3 — `argsMap` unification in `cmd/devforge-mcp`

- **File**: `cmd/devforge-mcp/main.go` (line 776) and `register_frontend.go` (line 255).
- **Change**: keep a single `argsMap` (or rename to `requestArgs` for clarity). Drop `frontendArgsMap`. If the original conflict was a signature mismatch, reconcile via an explicit signature instead of cloning.
- **Acceptance**: single definition of the args helper across `cmd/devforge-mcp/`. `go vet ./...` and `go build ./...` clean.

#### W4 — Description uplift for image / video / audio / Gemini tools

- **Files**: inline registrations in `cmd/devforge-mcp/main.go` for `generate_ui_image`, `ui2md`, `markdown_to_pdf`, `optimize_images`, `generate_favicon`, all `image_*` tools (lines ~287–470), all `video_*` tools, all `audio_*` tools.
- **Description style contract** (each tool's `mcp.WithDescription` MUST include):
  1. **Purpose** — one sentence, action verb first.
  2. **Key params** — explicit list of required/optional inputs with units (e.g. `width:int (pixels)`, `quality:int (1-100)`).
  3. **Example** — one concrete invocation as a comment or trailing sentence (`Example: trim audio from 00:00:05 to 00:00:30`).
  4. **Disambiguation** — when an adjacent tool exists (e.g. `image_palette` vs `image_placeholder`, `text_base64` vs `frontend_image_base64`), one short line: `Use X for ... ; use Y for ...`.
- **Out of bounds**: do NOT change tool names or parameter names. Only `mcp.WithDescription(...)` strings and parameter description strings.
- **Acceptance**: every image/video/audio/Gemini tool description is ≥ 2 sentences and contains an example. A reviewer reading only the descriptions can pick the right tool for "make a thumbnail of a video at 5 seconds" without reading source.

#### W5 — Context cancellation in pure-Go handlers

- **Files**: handlers under `internal/tools/{textenc,datafmt,cryptoutil,datetime,colors,frontend,backend,codetools}/` that currently signature-discard `ctx` via `_ context.Context`.
- **Change**: replace `_ context.Context` with `ctx context.Context`. In any loop body that processes more than ~1000 items or runs longer than a few ms in practice (the big offenders are `fake_data` row generation in `datafmt`, large CSV/JSON conversion, batch HMAC/hash operations, and `text_stats` over very long strings), add a `select { case <-ctx.Done(): return toolsutil.ErrResult("cancelled: " + ctx.Err().Error()); default: }` check at a coarse interval (every N iterations, not every iteration).
- **Out of bounds**: do not add context to trivially fast pure functions (single base64 encode, single regex run, single time conversion).
- **Acceptance**: a unit test in `datafmt/datafmt_test.go` cancels a synthetic 10k-row `fake_data` request mid-flight and asserts the error envelope contains `"cancelled"`.

#### W6 — DPF concurrency decision

- **File**: `internal/dpf/dpf.go` (`StreamClient`, lines 524–605).
- **Choice required** (this is a real fork — pick one in the design phase):
  - **Option A (Recommended)**: replace the single `StreamClient` mutex with a small fixed-size pool (e.g. 2–4 clients), each backed by its own dpf subprocess. Each call acquires a free client, sends, waits, releases. Memory cost: ~one dpf process per pool slot. Pool size configurable via `DEVFORGE_DPF_POOL_SIZE` env var, default 2.
  - **Option B**: keep the global mutex but document explicitly in `internal/dpf/dpf.go` that all dpf operations are serialized, add a comment with the rationale, and add a Prometheus-style counter exposed via a debug tool (out of scope if `metrics` are not desired — fall back to in-process logging only).
- **Acceptance for Option A**: a concurrent test that fires 8 parallel `image_resize` calls completes in ≤ 60% of the wall time of the serialized baseline. Pool exhaustion blocks the 9th caller rather than crashing.
- **Acceptance for Option B**: a comment at the top of `StreamClient` explains the invariant; no behavior changes.

#### W7 — Test coverage for media handlers

- **Create**: `internal/tools/image_tools_test.go`, `video_tools_test.go`, `audio_tools_test.go`, `optimize_images_test.go`, `generate_favicon_test.go`, `ui2md_test.go`.
- **Required cases per file**:
  - **Error path**: DPF-nil guard returns the canonical error envelope (now via `toolsutil.RequireDPF`).
  - **Happy path**: at least one case using a fake `dpf.StreamClient` (introduce a small interface in `internal/dpf` if needed — `type Streamer interface { Send(job dpf.Job) (dpf.Response, error) }` — and have `tools.Server` depend on the interface, with the real `StreamClient` satisfying it). Test injects a fake that returns canned responses.
- **Out of bounds**: do not require integration tests against the real dpf binary in CI.
- **Acceptance**: `go test ./internal/tools/...` passes. Coverage for the six listed files goes from ~0% to ≥ 50%.

### Sequencing & dependencies

W1 lands first (everything else builds on `toolsutil`). W2/W3/W4 are independent and can land in parallel. W5 depends on W1 (`ErrResult`). W6 (Option A) likely requires a tiny `Streamer` interface change that overlaps with W7 — coordinate by introducing the interface in W6 and using it in W7 tests.

### Files to create

- `internal/tools/toolsutil/toolsutil.go`
- `internal/tools/toolsutil/toolsutil_test.go`
- `internal/tools/image_tools_test.go`
- `internal/tools/video_tools_test.go`
- `internal/tools/audio_tools_test.go`
- `internal/tools/optimize_images_test.go`
- `internal/tools/generate_favicon_test.go`
- `internal/tools/ui2md_test.go`
- (if Option A) tests inside `internal/dpf/dpf_test.go` for pool behavior

### Files to modify

- All 13 handler packages under `internal/tools/*` to import `toolsutil` and drop local helpers
- `internal/tools/codetools/codetools.go` (hoist regexes)
- `cmd/devforge-mcp/main.go` (unify `argsMap`, uplift descriptions for inline image/video/audio/Gemini registrations)
- `cmd/devforge-mcp/register_frontend.go` (drop `frontendArgsMap`)
- `internal/dpf/dpf.go` (Option A: pool; Option B: invariant comment)
- `internal/tools/tools.go` (potentially swap `*dpf.StreamClient` for `dpf.Streamer` interface if Option A)

### Out of bounds for this PRP

- No new MCP tools.
- No changes to tool names, parameter names, or response shapes.
- No restructuring of `cmd/devforge-mcp/main.go` beyond `argsMap` unification and description edits (no file splits).
- No DB, no embeddings, no transport changes.
- No Rust-side changes to `dpf`.

### User stories

- **US-01**: As a DevForge contributor, when I add tool #89, I want to call `toolsutil.ErrResult` and `toolsutil.ResultJSON` instead of copying boilerplate from a neighbor file.
- **US-02**: As an LLM-driven agent calling DevForge, when I need to choose between `image_palette` and `image_placeholder`, the tool descriptions tell me unambiguously which one to call.
- **US-03**: As an MCP client invoking `code_metrics` on a large file, my call returns at least 30% faster than before because regex compilation no longer dominates.
- **US-04**: As an upstream caller that timed out my MCP request, my cancellation actually stops in-flight `fake_data` work instead of being ignored.
- **US-05**: As a CI engineer, when I run `go test ./...`, the media tool files have meaningful coverage instead of 0%.

### Acceptance criteria

- **AC-01**: `rg 'func errResult\\(' internal/tools/` returns exactly one hit (in `toolsutil`).
- **AC-02**: `rg 'func resultJSON\\(' internal/tools/` returns exactly one hit (in `toolsutil`).
- **AC-03**: `rg 'if s.DPF == nil' internal/tools/` returns zero hits.
- **AC-04**: `rg 'regexp.MustCompile' internal/tools/codetools/codetools.go` shows only package-level declarations (no calls inside functions for fixed patterns).
- **AC-05**: `rg 'frontendArgsMap' cmd/devforge-mcp/` returns zero hits.
- **AC-06**: A grep of `mcp.WithDescription` against image/video/audio/Gemini tools shows every description contains the word "Example" or an example fragment, and every adjacent-pair tool group includes a "Use X for ... use Y for ..." line on at least one side.
- **AC-07**: `go test ./...` passes with `CGO_ENABLED=0`.
- **AC-08**: Audit awk command from AGENTS.md still returns zero `WithArray` violations.
- **AC-09**: Manual smoke test: build `./dist/devforge-mcp`, attach via stdio, call `code_metrics` on a 500-line Go file and confirm response < previous baseline.
- **AC-10** (Option A only): a `TestPoolParallelism` in `internal/dpf/dpf_test.go` runs 8 fake-resize jobs in parallel and finishes in ≤ 60% of the serialized baseline wall time.
- **AC-11**: a cancellation test in `datafmt_test.go` proves `fake_data` returns a cancelled error envelope when its context is cancelled mid-flight.

---

## 4. Interaction

How Claude should behave during implementation:

- **Tone**: technical-direct. Comments only where the WHY is non-obvious (e.g. the pool size choice in W6 deserves a one-line comment; the regex hoist in W2 does not).
- **Allowed questions**: ONE upfront decision is required from the maintainer before W6 starts — **Option A (pool) vs Option B (documented serialization)**. Pause and ask before implementing. All other workstreams may proceed without questions.
- **Intermediate response format**: brief one-line updates per workstream completion. No paragraph summaries.
- **Language**: English for all code, comments, descriptions, tests, commit messages.
- **If a description rewrite is ambiguous**: prefer the more concrete example. Do not invent parameter behavior — read the handler source.
- **If a refactor would cause an MCP schema change**: STOP and surface to the maintainer. Schema-breaking changes are out of scope.
- **Commit cadence**: one work-unit commit per W-workstream. Use conventional commits: `refactor(tools):`, `perf(codetools):`, `docs(mcp):`, `test(tools):`, `feat(dpf):` (pool only).

---

## 5. Response

Expected output structure at the end of the session:

### Code artifacts

- `internal/tools/toolsutil/toolsutil.go` — shared `ErrResult`, `ResultJSON`, `RequireDPF`
- `internal/tools/toolsutil/toolsutil_test.go` — table-driven tests for all three helpers
- 13 handler packages updated to import `toolsutil` and drop local duplicates
- `internal/tools/codetools/codetools.go` — package-level cached regexes; bench file
- `cmd/devforge-mcp/main.go` — unified `argsMap`; uplifted descriptions on all inline image/video/audio/Gemini registrations
- `cmd/devforge-mcp/register_frontend.go` — `frontendArgsMap` removed
- `internal/dpf/dpf.go` — pool implementation (Option A) OR invariant comment (Option B)
- 6 new test files under `internal/tools/` covering previously-untested media handlers

### Quality Gates (all must pass before considering complete)

- [ ] `CGO_ENABLED=0 go build ./...` clean
- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean
- [ ] AGENTS.md `WithArray` audit awk command returns zero violations
- [ ] `rg 'func errResult\\(' internal/tools/ | wc -l` returns 1
- [ ] `rg 'func resultJSON\\(' internal/tools/ | wc -l` returns 1
- [ ] `rg 'if s.DPF == nil' internal/tools/ | wc -l` returns 0
- [ ] `go test -bench=. ./internal/tools/codetools/` shows non-regression with cached regexes
- [ ] (Option A) DPF pool parallelism test passes
- [ ] Manual smoke test: stdio session with one image, one video, one audio, one text tool call — each returns successfully
- [ ] PRP completion notes record the W6 decision (A vs B) and the benchmark numbers from W2

### Documentation to update

- `AGENTS.md` — add a "Shared helpers" subsection pointing at `internal/tools/toolsutil` and the description style contract.
- This PRP — status moves to `Done` only after all Quality Gates pass.
- `CHANGELOG.md` if it exists (otherwise skip).

### Do not include

- New MCP tools.
- Changes to existing tool names, parameter names, or response JSON shapes.
- Restructuring of `main.go` beyond the explicit edits listed in W3 and W4.
- DB or embedding reintroductions.
- Rust-side dpf changes.
- Unrelated description rewrites for text/datetime/backend tools (their descriptions are already adequate — do not gold-plate).

---

## Decisions resolved during implementation

- **D1 — DPF concurrency**: ✅ **Option A** (pool of StreamClients) accepted by maintainer on 2026-05-24.
- **D2 — Pool size default**: pending — agent must pick a sensible default (2) and expose `DEVFORGE_DPF_POOL_SIZE` env var.
- **D3 — Streamer interface**: ✅ Introduce `dpf.Streamer` interface as part of W6. W7 depends on it for fake injection.
- **W5 scope adjustment**: ✅ `fake_data` row cap raised from 100 → 10_000 accepted by maintainer on 2026-05-24. Rationale: 100 was too restrictive for a fake-data generator and 10_000 is large enough to exercise the cancellation path in the W5 test. Documented here so the change is intentional, not silent.
