# Code Utilities Tools (`codetools`)

Tools for formatting, analyzing, and rendering source code. All tools are stateless and require no external dependencies beyond the Go standard library.

---

## `code_format`

Format source code for a given language.

**Supported languages:**
- `go` — uses `go/format.Source()`, always tab-indented per Go standard
- `json` — re-indents with configurable indent (spaces or tabs)
- `typescript` (also accepts `javascript`) — indent normalization; detects current indent unit and re-indents to target
- `html` — heuristic tag indentation: one tag per line, children indented
- `css` — normalizes rule blocks: one property per line, consistent indentation

### Input

| Parameter    | Type    | Required | Default | Description |
|-------------|---------|----------|---------|-------------|
| `code`       | string  | ✓        | —       | Source code to format |
| `language`   | string  | ✓        | —       | `go \| typescript \| json \| html \| css` |
| `indent_size`| integer | —        | `2`     | Indent size in spaces (ignored for Go) |
| `use_tabs`   | boolean | —        | `false` | Use tabs instead of spaces (Go always uses tabs) |

### Output

```json
{
  "result": "<formatted code>",
  "language": "<lang>",
  "changed": true
}
```

| Field      | Type    | Description |
|-----------|---------|-------------|
| `result`   | string  | Formatted source code |
| `language` | string  | Normalized language identifier |
| `changed`  | boolean | `true` if the output differs from the input |

### Error responses

| Condition | Error message |
|-----------|---------------|
| `code` is empty or whitespace | `"code is required"` |
| Unsupported language | `"unsupported language \"X\" (supported: ...)"` |
| Go syntax error | `"Go format error: ..."` |
| Invalid JSON | `"JSON format error: ..."` |

### Examples

**Format Go code:**
```json
{
  "code": "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"hello\")}",
  "language": "go"
}
```

**Format JSON with 4-space indent:**
```json
{
  "code": "{\"a\":1,\"b\":[1,2]}",
  "language": "json",
  "indent_size": 4
}
```

**Re-indent TypeScript from 4 spaces to 2:**
```json
{
  "code": "function hello() {\n    const x = 1;\n    return x;\n}",
  "language": "typescript",
  "indent_size": 2
}
```

---

## `code_metrics`

Compute code quality metrics for source code.

### Input

| Parameter  | Type   | Required | Description |
|-----------|--------|----------|-------------|
| `code`     | string | ✓        | Source code to analyze |
| `language` | string | ✓        | `go \| typescript \| python \| generic` |

### Output

```json
{
  "loc": 42,
  "sloc": 30,
  "blank_lines": 8,
  "comment_lines": 4,
  "functions": 5,
  "complexity_estimate": 12,
  "language": "go"
}
```

| Field                | Type    | Description |
|---------------------|---------|-------------|
| `loc`                | integer | Total lines including blank and comments |
| `sloc`               | integer | Source lines (non-blank, non-comment) |
| `blank_lines`        | integer | Lines that are empty or only whitespace |
| `comment_lines`      | integer | Lines that are comments (`//`, `#`, `/* */`, docstrings) |
| `functions`          | integer | Count of function/method declarations |
| `complexity_estimate`| integer | Count of branching points |
| `language`           | string  | Normalized language identifier |

**Invariant:** `loc = sloc + blank_lines + comment_lines`

### Complexity definition

Counted branching points per language:

| Token | Go | TypeScript | Python |
|-------|----|-----------:|-------:|
| `if` / `elif` | ✓ | ✓ | ✓ |
| `else if` | — | ✓ | — |
| `for` / `while` | ✓ | ✓ | ✓ |
| `case` (switch) | ✓ | ✓ | ✓ |
| `select` | ✓ | — | — |
| `&&` / `\|\|` | ✓ | ✓ | — |
| `and` / `or` | — | — | ✓ |
| `?` (ternary) | — | ✓ | — |

For **Go**, function and complexity counts use `go/ast` when the source is valid Go; otherwise falls back to regex.

### Error responses

| Condition | Error message |
|-----------|---------------|
| `code` is empty or whitespace | `"code is required"` |
| Unsupported language | `"unsupported language \"X\" (supported: ...)"` |

### Example

```json
{
  "code": "package main\n\nfunc add(a, b int) int {\n\tif a > 0 {\n\t\treturn a + b\n\t}\n\treturn b\n}",
  "language": "go"
}
```

Response:
```json
{
  "loc": 8,
  "sloc": 7,
  "blank_lines": 1,
  "comment_lines": 0,
  "functions": 1,
  "complexity_estimate": 1,
  "language": "go"
}
```

---

## `code_template`

Render a template string with JSON context bindings.

### Input

| Parameter  | Type   | Required | Default | Description |
|-----------|--------|----------|---------|-------------|
| `template` | string | ✓        | —       | Template string to render |
| `context`  | string | ✓        | —       | JSON object with variable bindings |
| `engine`   | string | —        | `go`    | Template engine: `go \| mustache` |

### Output

```json
{
  "result": "<rendered output>"
}
```

### Go engine (`text/template`)

Uses Go's standard `text/template` package. Full template syntax is supported:
- `{{.variable}}` — variable access
- `{{range .items}}...{{end}}` — iteration
- `{{if .condition}}...{{end}}` — conditional
- `{{with .value}}...{{end}}` — scope change
- All built-in functions (`len`, `index`, `printf`, etc.)

**Security note:** Uses `text/template`, not `html/template`. Callers control escaping.

### Mustache engine (built-in minimal interpreter)

Supported Mustache tags:

| Tag | Description |
|-----|-------------|
| `{{variable}}` | Variable interpolation |
| `{{{variable}}}` | Unescaped variable (same behavior) |
| `{{#section}}...{{/section}}` | Truthy check; iterates if value is an array |
| `{{^inverted}}...{{/inverted}}` | Renders when value is falsy or empty |
| `{{! comment }}` | Comment (stripped from output) |

**Dot notation** is supported for nested values: `{{user.name}}`.

**Iteration:** When a section value is an array of objects, each item's keys are merged into the context for that iteration.

### Error responses

| Condition | Error message |
|-----------|---------------|
| `template` is empty | `"template is required"` |
| `context` is empty | `"context is required"` |
| Invalid context JSON | `"invalid context JSON: ..."` |
| Go template parse error | `"template execution error: parse error: ..."` |
| Go template execute error | `"template execution error: execute error: ..."` |
| Mustache parse/render error | `"mustache error: ..."` |
| Unsupported engine | `"unsupported engine \"X\" (supported: go, mustache)"` |

### Examples

**Go template with range:**
```json
{
  "template": "{{range .items}}- {{.}}\n{{end}}",
  "context": "{\"items\": [\"foo\", \"bar\", \"baz\"]}",
  "engine": "go"
}
```

Output: `- foo\n- bar\n- baz\n`

**Mustache with section and iteration:**
```json
{
  "template": "{{#users}}Hello, {{name}}!\n{{/users}}",
  "context": "{\"users\": [{\"name\": \"Alice\"}, {\"name\": \"Bob\"}]}",
  "engine": "mustache"
}
```

Output: `Hello, Alice!\nHello, Bob!\n`

**Mustache inverted section:**
```json
{
  "template": "{{^items}}No items found.{{/items}}",
  "context": "{\"items\": []}",
  "engine": "mustache"
}
```

Output: `No items found.`

---

## Implementation Notes

- **Package**: `dev-forge-mcp/internal/tools/codetools`
- **Registration**: `cmd/devforge-mcp/register_codetools.go`
- **Dependencies**: Standard library only (`go/format`, `go/parser`, `go/ast`, `go/token`, `text/template`, `regexp`, `strings`, `unicode`)
- **No external formatters** — no Prettier, no goimports beyond the stdlib `go/format`
- All tools accept `context.Context` as first argument
- All errors return `{"error": "message"}` — tools never panic
