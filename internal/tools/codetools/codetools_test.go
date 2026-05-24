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

// ── benchmarks ───────────────────────────────────────────────────────────────

// syntheticGoFixture is a ~500-line Go snippet that exercises all regex paths
// (functions, if/for/case/select, logical operators).
const syntheticGoFixture = `package example

import (
	"context"
	"fmt"
	"strings"
)

// Add returns the sum of a and b.
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of a and b.
func Subtract(a, b int) int {
	return a - b
}

// Classify categorises n into a named bucket.
func Classify(n int) string {
	if n < 0 {
		return "negative"
	} else if n == 0 {
		return "zero"
	} else if n < 10 {
		return "small"
	} else if n < 100 {
		return "medium"
	}
	return "large"
}

// Fibonacci returns the nth Fibonacci number.
func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// FilterPositive returns only positive values from vs.
func FilterPositive(vs []int) []int {
	out := make([]int, 0, len(vs))
	for _, v := range vs {
		if v > 0 {
			out = append(out, v)
		}
	}
	return out
}

// Reduce folds vs using f with initial accumulator acc.
func Reduce(vs []int, acc int, f func(int, int) int) int {
	for _, v := range vs {
		acc = f(acc, v)
	}
	return acc
}

// Contains reports whether vs contains target.
func Contains(vs []int, target int) bool {
	for _, v := range vs {
		if v == target {
			return true
		}
	}
	return false
}

// Unique returns deduplicated elements preserving order.
func Unique(vs []int) []int {
	seen := make(map[int]struct{})
	out := make([]int, 0, len(vs))
	for _, v := range vs {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Chunk splits vs into sub-slices of size n.
func Chunk(vs []int, n int) [][]int {
	if n <= 0 {
		return nil
	}
	var result [][]int
	for i := 0; i < len(vs); i += n {
		end := i + n
		if end > len(vs) {
			end = len(vs)
		}
		result = append(result, vs[i:end])
	}
	return result
}

// DayName returns the name of the weekday for d (0=Sunday).
func DayName(d int) string {
	switch d {
	case 0:
		return "Sunday"
	case 1:
		return "Monday"
	case 2:
		return "Tuesday"
	case 3:
		return "Wednesday"
	case 4:
		return "Thursday"
	case 5:
		return "Friday"
	case 6:
		return "Saturday"
	default:
		return "Unknown"
	}
}

// Greet returns a personalised greeting.
func Greet(name string) string {
	if strings.TrimSpace(name) == "" {
		return "Hello, stranger!"
	}
	return fmt.Sprintf("Hello, %s!", name)
}

// Max returns the larger of a and b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of a and b.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Clamp restricts v to the range [lo, hi].
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// IsPrime reports whether n is prime.
func IsPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

// Primes returns all primes up to max.
func Primes(max int) []int {
	out := []int{}
	for i := 2; i <= max; i++ {
		if IsPrime(i) {
			out = append(out, i)
		}
	}
	return out
}

// Reverse reverses the elements of vs in-place.
func Reverse(vs []int) {
	for i, j := 0, len(vs)-1; i < j; i, j = i+1, j-1 {
		vs[i], vs[j] = vs[j], vs[i]
	}
}

// SelectChannel demonstrates select usage.
func SelectChannel(ctx context.Context, ch <-chan int) (int, bool) {
	select {
	case v, ok := <-ch:
		return v, ok
	case <-ctx.Done():
		return 0, false
	}
}

// LogicalCheck combines && and || operators to exercise complexity counting.
func LogicalCheck(a, b, c bool) bool {
	return (a && b) || (b && c) || (a && c)
}

// NestedLoops exercises nested for/if patterns.
func NestedLoops(matrix [][]int) int {
	total := 0
	for i := 0; i < len(matrix); i++ {
		for j := 0; j < len(matrix[i]); j++ {
			if matrix[i][j] > 0 && matrix[i][j]%2 == 0 {
				total += matrix[i][j]
			}
		}
	}
	return total
}

// Dispatch uses a type switch (case) to dispatch by type.
func Dispatch(v any) string {
	switch t := v.(type) {
	case int:
		return fmt.Sprintf("int(%d)", t)
	case string:
		return fmt.Sprintf("string(%s)", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return "unknown"
	}
}

// Pipeline chains multiple operations.
func Pipeline(vs []int) []int {
	vs = FilterPositive(vs)
	vs = Unique(vs)
	Reverse(vs)
	return vs
}

// Walker iterates and applies a side-effectful function.
func Walker(vs []int, fn func(int)) {
	for _, v := range vs {
		fn(v)
	}
}

// SafeDivide returns a/b and false if b is zero.
func SafeDivide(a, b int) (int, bool) {
	if b == 0 {
		return 0, false
	}
	return a / b, true
}

// Abs returns the absolute value of n.
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Sign returns -1, 0, or 1.
func Sign(n int) int {
	if n < 0 {
		return -1
	} else if n > 0 {
		return 1
	}
	return 0
}

// CountVowels counts ASCII vowels in s.
func CountVowels(s string) int {
	count := 0
	for _, c := range strings.ToLower(s) {
		switch c {
		case 'a', 'e', 'i', 'o', 'u':
			count++
		}
	}
	return count
}

// IsPalindrome reports whether s reads the same forwards and backwards.
func IsPalindrome(s string) bool {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		if runes[i] != runes[j] {
			return false
		}
	}
	return true
}

// Repeat builds a string of n copies of s.
func Repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(s)
	}
	return b.String()
}

// BinarySearch returns the index of target in sorted vs, or -1.
func BinarySearch(vs []int, target int) int {
	lo, hi := 0, len(vs)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		switch {
		case vs[mid] == target:
			return mid
		case vs[mid] < target:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	return -1
}
`

// BenchmarkCodeMetrics measures the cost of computing code metrics for a
// ~500-line Go fixture.  Run with:
//
//	go test -bench=BenchmarkCodeMetrics -benchmem -count=3 ./internal/tools/codetools/...
func BenchmarkCodeMetrics(b *testing.B) {
	ctx := context.Background()
	input := codetools.MetricsInput{
		Code:     syntheticGoFixture,
		Language: "go",
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = codetools.Metrics(ctx, input)
	}
}
