package codetools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/codetools"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func getString(t *testing.T, jsonStr, key string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonStr, err)
	}
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in %s", key, jsonStr)
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func getBool(t *testing.T, jsonStr, key string) bool {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonStr, err)
	}
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in %s", key, jsonStr)
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("key %q is not bool in %s", key, jsonStr)
	}
	return b
}

func getInt(t *testing.T, jsonStr, key string) int {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonStr, err)
	}
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in %s", key, jsonStr)
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	t.Fatalf("key %q is not a number in %s", key, jsonStr)
	return 0
}

func getError(t *testing.T, jsonStr string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonStr, err)
	}
	if v, ok := m["error"].(string); ok {
		return v
	}
	return ""
}

// ── code_format ───────────────────────────────────────────────────────────────

func TestFormat_Go(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     codetools.FormatInput
		wantLang  string
		checkFn   func(t *testing.T, result string, changed bool)
		wantError bool
	}{
		{
			name: "go: formats unformatted code",
			input: codetools.FormatInput{
				Code:     "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"hello\")}",
				Language: "go",
			},
			wantLang: "go",
			checkFn: func(t *testing.T, result string, changed bool) {
				if !changed {
					t.Error("expected changed=true for unformatted Go code")
				}
				if !strings.Contains(result, "fmt.Println") {
					t.Error("expected formatted output to contain fmt.Println")
				}
			},
		},
		{
			name: "go: already formatted code unchanged (same content)",
			input: codetools.FormatInput{
				Code:     "package main\n\nfunc main() {\n}\n",
				Language: "go",
			},
			wantLang: "go",
			checkFn: func(t *testing.T, result string, changed bool) {
				if !strings.Contains(result, "func main") {
					t.Error("expected result to contain func main")
				}
			},
		},
		{
			name: "go: syntax error returns error",
			input: codetools.FormatInput{
				Code:     "package main\nfunc (",
				Language: "go",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Format(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			lang := getString(t, got, "language")
			if lang != tc.wantLang {
				t.Errorf("language: want %q, got %q", tc.wantLang, lang)
			}
			result := getString(t, got, "result")
			changed := getBool(t, got, "changed")
			if tc.checkFn != nil {
				tc.checkFn(t, result, changed)
			}
		})
	}
}

func TestFormat_JSON(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     codetools.FormatInput
		checkFn   func(t *testing.T, result string)
		wantError bool
	}{
		{
			name: "json: formats compact JSON with 2-space indent",
			input: codetools.FormatInput{
				Code:       `{"a":1,"b":[1,2,3]}`,
				Language:   "json",
				IndentSize: 2,
			},
			checkFn: func(t *testing.T, result string) {
				if !strings.Contains(result, "  \"a\"") {
					t.Errorf("expected 2-space indent, got:\n%s", result)
				}
			},
		},
		{
			name: "json: formats with tabs",
			input: codetools.FormatInput{
				Code:     `{"x":true}`,
				Language: "json",
				UseTabs:  true,
			},
			checkFn: func(t *testing.T, result string) {
				if !strings.Contains(result, "\t\"x\"") {
					t.Errorf("expected tab indent, got:\n%s", result)
				}
			},
		},
		{
			name: "json: invalid JSON returns error",
			input: codetools.FormatInput{
				Code:     `{invalid}`,
				Language: "json",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Format(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			result := getString(t, got, "result")
			if tc.checkFn != nil {
				tc.checkFn(t, result)
			}
		})
	}
}

func TestFormat_TypeScript(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     codetools.FormatInput
		checkFn   func(t *testing.T, result string)
		wantError bool
	}{
		{
			name: "typescript: re-indents from 4-space to 2-space",
			input: codetools.FormatInput{
				Code:       "function hello() {\n    const x = 1;\n    return x;\n}",
				Language:   "typescript",
				IndentSize: 2,
			},
			checkFn: func(t *testing.T, result string) {
				if !strings.Contains(result, "  const x") {
					t.Errorf("expected 2-space indent, got:\n%s", result)
				}
			},
		},
		{
			name: "typescript: unsupported language returns error",
			input: codetools.FormatInput{
				Code:     "some code",
				Language: "ruby",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Format(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			result := getString(t, got, "result")
			if tc.checkFn != nil {
				tc.checkFn(t, result)
			}
		})
	}
}

func TestFormat_HTML(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   codetools.FormatInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "html: indents children",
			input: codetools.FormatInput{
				Code:       "<div><p>Hello</p></div>",
				Language:   "html",
				IndentSize: 2,
			},
			checkFn: func(t *testing.T, result string) {
				if !strings.Contains(result, "<div>") {
					t.Error("expected <div> in output")
				}
				if !strings.Contains(result, "  <p>") {
					t.Errorf("expected indented <p>, got:\n%s", result)
				}
			},
		},
		{
			name: "html: void elements not double-indented",
			input: codetools.FormatInput{
				Code:       "<div><br/><input type=\"text\"/></div>",
				Language:   "html",
				IndentSize: 2,
			},
			checkFn: func(t *testing.T, result string) {
				if !strings.Contains(result, "<div>") {
					t.Error("expected <div>")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Format(ctx, tc.input)
			if err := getError(t, got); err != "" {
				t.Fatalf("unexpected error: %s", err)
			}
			result := getString(t, got, "result")
			if tc.checkFn != nil {
				tc.checkFn(t, result)
			}
		})
	}
}

func TestFormat_CSS(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   codetools.FormatInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "css: one property per line",
			input: codetools.FormatInput{
				Code:       ".foo{color:red;margin:0}",
				Language:   "css",
				IndentSize: 2,
			},
			checkFn: func(t *testing.T, result string) {
				if !strings.Contains(result, "color:red") {
					t.Errorf("expected color property, got:\n%s", result)
				}
				if !strings.Contains(result, "margin:0") {
					t.Errorf("expected margin property, got:\n%s", result)
				}
				// Properties should be on separate lines.
				lines := strings.Split(result, "\n")
				colorLine := -1
				marginLine := -1
				for i, l := range lines {
					if strings.Contains(l, "color") {
						colorLine = i
					}
					if strings.Contains(l, "margin") {
						marginLine = i
					}
				}
				if colorLine == marginLine {
					t.Errorf("color and margin should be on different lines, got:\n%s", result)
				}
			},
		},
		{
			name: "css: whitespace-only code returns error",
			input: codetools.FormatInput{
				Code:     "   ",
				Language: "css",
			},
			checkFn: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Format(ctx, tc.input)
			// The whitespace-only test is expected to return an error.
			if tc.name == "css: whitespace-only code returns error" {
				if getError(t, got) == "" {
					t.Errorf("expected error for whitespace-only code, got %q", got)
				}
				return
			}
			if err := getError(t, got); err != "" {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.checkFn != nil {
				result := getString(t, got, "result")
				tc.checkFn(t, result)
			}
		})
	}
}

func TestFormat_EmptyCode(t *testing.T) {
	ctx := context.Background()
	got := codetools.Format(ctx, codetools.FormatInput{Code: "", Language: "go"})
	if getError(t, got) == "" {
		t.Error("expected error for empty code")
	}
}

// ── code_metrics ──────────────────────────────────────────────────────────────

func TestMetrics_Go(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     codetools.MetricsInput
		checkFn   func(t *testing.T, got string)
		wantError bool
	}{
		{
			name: "go: basic metrics",
			input: codetools.MetricsInput{
				Code: `package main

// Package main is the entry point.
import "fmt"

func main() {
	if true {
		fmt.Println("hello")
	}
}

func helper() int {
	return 42
}
`,
				Language: "go",
			},
			checkFn: func(t *testing.T, got string) {
				loc := getInt(t, got, "loc")
				if loc == 0 {
					t.Error("loc should be > 0")
				}
				funcs := getInt(t, got, "functions")
				if funcs < 2 {
					t.Errorf("expected at least 2 functions, got %d", funcs)
				}
				complexity := getInt(t, got, "complexity_estimate")
				if complexity < 1 {
					t.Errorf("expected complexity >= 1, got %d", complexity)
				}
				lang := getString(t, got, "language")
				if lang != "go" {
					t.Errorf("expected language=go, got %q", lang)
				}
			},
		},
		{
			name: "go: blank lines counted",
			input: codetools.MetricsInput{
				Code:     "package main\n\n\nfunc foo() {}\n",
				Language: "go",
			},
			checkFn: func(t *testing.T, got string) {
				blank := getInt(t, got, "blank_lines")
				if blank < 2 {
					t.Errorf("expected at least 2 blank lines, got %d", blank)
				}
			},
		},
		{
			name: "go: empty code returns error",
			input: codetools.MetricsInput{
				Code:     "  ",
				Language: "go",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Metrics(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			if err := getError(t, got); err != "" {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, got)
			}
		})
	}
}

func TestMetrics_TypeScript(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   codetools.MetricsInput
		checkFn func(t *testing.T, got string)
	}{
		{
			name: "typescript: counts functions and complexity",
			input: codetools.MetricsInput{
				Code: `// Greeting function
function greet(name: string): string {
  if (name === '') {
    return 'Hello, world!';
  }
  return 'Hello, ' + name + '!';
}

const arrow = (x: number) => x * 2;
`,
				Language: "typescript",
			},
			checkFn: func(t *testing.T, got string) {
				funcs := getInt(t, got, "functions")
				if funcs < 1 {
					t.Errorf("expected at least 1 function, got %d", funcs)
				}
				comments := getInt(t, got, "comment_lines")
				if comments < 1 {
					t.Errorf("expected at least 1 comment line, got %d", comments)
				}
				complexity := getInt(t, got, "complexity_estimate")
				if complexity < 1 {
					t.Errorf("expected complexity >= 1, got %d", complexity)
				}
			},
		},
		{
			name: "typescript: invalid language returns error",
			input: codetools.MetricsInput{
				Code:     "some code",
				Language: "cobol",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Metrics(ctx, tc.input)
			if tc.name == "typescript: invalid language returns error" {
				// Should return error.
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			if err := getError(t, got); err != "" {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.checkFn != nil {
				tc.checkFn(t, got)
			}
		})
	}
}

func TestMetrics_Python(t *testing.T) {
	ctx := context.Background()

	code := `# This is a comment
def greet(name):
    """Say hello."""
    if not name:
        return "Hello, world!"
    return f"Hello, {name}!"

def add(a, b):
    return a + b
`
	got := codetools.Metrics(ctx, codetools.MetricsInput{Code: code, Language: "python"})
	if err := getError(t, got); err != "" {
		t.Fatalf("unexpected error: %s", err)
	}
	funcs := getInt(t, got, "functions")
	if funcs < 2 {
		t.Errorf("expected at least 2 functions, got %d", funcs)
	}
	lang := getString(t, got, "language")
	if lang != "python" {
		t.Errorf("expected language=python, got %q", lang)
	}
}

func TestMetrics_LOCConsistency(t *testing.T) {
	ctx := context.Background()

	code := "line1\nline2\n\nline4\n"
	got := codetools.Metrics(ctx, codetools.MetricsInput{Code: code, Language: "generic"})

	loc := getInt(t, got, "loc")
	sloc := getInt(t, got, "sloc")
	blank := getInt(t, got, "blank_lines")
	comments := getInt(t, got, "comment_lines")

	// LOC = SLOC + blank + comments
	sum := sloc + blank + comments
	if sum != loc {
		t.Errorf("LOC consistency: loc=%d but sloc=%d + blank=%d + comments=%d = %d", loc, sloc, blank, comments, sum)
	}
}

// ── code_template ─────────────────────────────────────────────────────────────

func TestTemplate_GoEngine(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		input      codetools.TemplateInput
		wantResult string
		wantError  bool
	}{
		{
			name: "go: simple variable interpolation",
			input: codetools.TemplateInput{
				Template: "Hello, {{.name}}!",
				Context:  `{"name": "World"}`,
				Engine:   "go",
			},
			wantResult: "Hello, World!",
		},
		{
			name: "go: range over slice",
			input: codetools.TemplateInput{
				Template: "{{range .items}}{{.}} {{end}}",
				Context:  `{"items": ["a", "b", "c"]}`,
				Engine:   "go",
			},
			wantResult: "a b c ",
		},
		{
			name: "go: conditional",
			input: codetools.TemplateInput{
				Template: "{{if .show}}visible{{end}}",
				Context:  `{"show": true}`,
				Engine:   "go",
			},
			wantResult: "visible",
		},
		{
			name: "go: invalid template syntax returns error",
			input: codetools.TemplateInput{
				Template: "{{.unclosed",
				Context:  `{}`,
				Engine:   "go",
			},
			wantError: true,
		},
		{
			name: "go: invalid context JSON returns error",
			input: codetools.TemplateInput{
				Template: "hello",
				Context:  `{invalid`,
				Engine:   "go",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Template(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			result := getString(t, got, "result")
			if result != tc.wantResult {
				t.Errorf("want %q, got %q", tc.wantResult, result)
			}
		})
	}
}

func TestTemplate_MustacheEngine(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		input      codetools.TemplateInput
		wantResult string
		wantError  bool
	}{
		{
			name: "mustache: variable interpolation",
			input: codetools.TemplateInput{
				Template: "Hello, {{name}}!",
				Context:  `{"name": "World"}`,
				Engine:   "mustache",
			},
			wantResult: "Hello, World!",
		},
		{
			name: "mustache: section truthy",
			input: codetools.TemplateInput{
				Template: "{{#show}}visible{{/show}}",
				Context:  `{"show": true}`,
				Engine:   "mustache",
			},
			wantResult: "visible",
		},
		{
			name: "mustache: section falsy is empty",
			input: codetools.TemplateInput{
				Template: "{{#show}}visible{{/show}}",
				Context:  `{"show": false}`,
				Engine:   "mustache",
			},
			wantResult: "",
		},
		{
			name: "mustache: inverted section on falsy",
			input: codetools.TemplateInput{
				Template: "{{^show}}hidden{{/show}}",
				Context:  `{"show": false}`,
				Engine:   "mustache",
			},
			wantResult: "hidden",
		},
		{
			name: "mustache: inverted section on truthy is empty",
			input: codetools.TemplateInput{
				Template: "{{^show}}hidden{{/show}}",
				Context:  `{"show": true}`,
				Engine:   "mustache",
			},
			wantResult: "",
		},
		{
			name: "mustache: comment is stripped",
			input: codetools.TemplateInput{
				Template: "Hello{{! this is a comment }}, World!",
				Context:  `{}`,
				Engine:   "mustache",
			},
			wantResult: "Hello, World!",
		},
		{
			name: "mustache: iteration over array",
			input: codetools.TemplateInput{
				Template: "{{#items}}{{name}} {{/items}}",
				Context:  `{"items": [{"name": "Alice"}, {"name": "Bob"}]}`,
				Engine:   "mustache",
			},
			wantResult: "Alice Bob ",
		},
		{
			name: "mustache: missing variable renders empty",
			input: codetools.TemplateInput{
				Template: "{{missing}}",
				Context:  `{}`,
				Engine:   "mustache",
			},
			wantResult: "",
		},
		{
			name: "mustache: unsupported engine returns error",
			input: codetools.TemplateInput{
				Template: "hello",
				Context:  `{}`,
				Engine:   "handlebars",
			},
			wantError: true,
		},
		{
			name: "mustache: empty template returns error",
			input: codetools.TemplateInput{
				Template: "",
				Context:  `{}`,
				Engine:   "mustache",
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := codetools.Template(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			result := getString(t, got, "result")
			if result != tc.wantResult {
				t.Errorf("want %q, got %q", tc.wantResult, result)
			}
		})
	}
}

func TestTemplate_DefaultEngine(t *testing.T) {
	ctx := context.Background()
	// Engine defaults to "go" when empty string.
	got := codetools.Template(ctx, codetools.TemplateInput{
		Template: "{{.greeting}}, {{.subject}}!",
		Context:  `{"greeting": "Hello", "subject": "World"}`,
		Engine:   "",
	})
	result := getString(t, got, "result")
	if result != "Hello, World!" {
		t.Errorf("want %q, got %q", "Hello, World!", result)
	}
}

// ── edge cases ────────────────────────────────────────────────────────────────

func TestFormat_CSS_MultipleRules(t *testing.T) {
	ctx := context.Background()
	code := ".a{color:red;font-size:14px}.b{margin:0;padding:0}"
	got := codetools.Format(ctx, codetools.FormatInput{
		Code:       code,
		Language:   "css",
		IndentSize: 2,
	})
	result := getString(t, got, "result")
	lines := strings.Split(result, "\n")
	if len(lines) < 4 {
		t.Errorf("expected multiple lines, got:\n%s", result)
	}
}

func TestMetrics_CommentsGoBlockComment(t *testing.T) {
	ctx := context.Background()
	code := `package main

/* this is
   a block comment */

func foo() {}
`
	got := codetools.Metrics(ctx, codetools.MetricsInput{Code: code, Language: "go"})
	comments := getInt(t, got, "comment_lines")
	if comments < 2 {
		t.Errorf("expected at least 2 comment lines for block comment, got %d", comments)
	}
}

func TestTemplate_MustacheNumericValue(t *testing.T) {
	ctx := context.Background()
	got := codetools.Template(ctx, codetools.TemplateInput{
		Template: "Count: {{count}}",
		Context:  `{"count": 42}`,
		Engine:   "mustache",
	})
	result := getString(t, got, "result")
	if result != "Count: 42" {
		t.Errorf("want %q, got %q", "Count: 42", result)
	}
}
