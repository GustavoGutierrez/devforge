// Package codetools implements MCP tools for code utilities.
// Tools: code_format, code_metrics, code_template.
package codetools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

// errResult returns a JSON-encoded error response.
func errResult(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

// resultJSON marshals v to JSON or returns an error JSON.
func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errResult("marshal failed: " + err.Error())
	}
	return string(b)
}

// ─── code_format ─────────────────────────────────────────────────────────────

// FormatInput is the input schema for the code_format tool.
type FormatInput struct {
	Code       string `json:"code"`
	Language   string `json:"language"`    // go | typescript | json | html | css
	IndentSize int    `json:"indent_size"` // default 2
	UseTabs    bool   `json:"use_tabs"`    // default false (except Go always uses tabs)
}

// FormatOutput is the output schema for the code_format tool.
type FormatOutput struct {
	Result   string `json:"result"`
	Language string `json:"language"`
	Changed  bool   `json:"changed"`
}

// Format formats source code for the given language.
func Format(_ context.Context, input FormatInput) string {
	if strings.TrimSpace(input.Code) == "" {
		return errResult("code is required")
	}

	lang := strings.ToLower(strings.TrimSpace(input.Language))
	indentSize := input.IndentSize
	if indentSize <= 0 {
		indentSize = 2
	}

	var result string
	var err error

	switch lang {
	case "go":
		result, err = formatGo(input.Code)
		if err != nil {
			return errResult("Go format error: " + err.Error())
		}

	case "json":
		result, err = formatJSON(input.Code, indentSize, input.UseTabs)
		if err != nil {
			return errResult("JSON format error: " + err.Error())
		}

	case "typescript", "javascript", "ts", "js":
		result = formatIndentBased(input.Code, indentSize, input.UseTabs)
		lang = "typescript"

	case "html":
		result = formatHTML(input.Code, indentSize, input.UseTabs)

	case "css":
		result = formatCSS(input.Code, indentSize, input.UseTabs)

	default:
		return errResult(fmt.Sprintf("unsupported language %q (supported: go, typescript, json, html, css)", input.Language))
	}

	return resultJSON(FormatOutput{
		Result:   result,
		Language: lang,
		Changed:  result != input.Code,
	})
}

// formatGo formats Go source using go/format.Source (always tab-indented).
func formatGo(code string) (string, error) {
	out, err := format.Source([]byte(code))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// formatJSON re-indents a JSON document using the requested indent style.
func formatJSON(code string, indentSize int, useTabs bool) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(code), &v); err != nil {
		return "", err
	}
	var indent string
	if useTabs {
		indent = "\t"
	} else {
		indent = strings.Repeat(" ", indentSize)
	}
	out, err := json.MarshalIndent(v, "", indent)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// formatIndentBased performs simple indent normalization for TypeScript/JavaScript.
// It detects the current indent unit, then re-indents each line to match
// the requested target style.
func formatIndentBased(code string, indentSize int, useTabs bool) string {
	lines := strings.Split(code, "\n")

	// Detect the current indent unit (smallest non-zero leading whitespace).
	currentUnit := detectIndentUnit(lines)

	// Build target indent string.
	var targetUnit string
	if useTabs {
		targetUnit = "\t"
	} else {
		targetUnit = strings.Repeat(" ", indentSize)
	}

	if currentUnit == "" || currentUnit == targetUnit {
		return code
	}

	var out []string
	for _, line := range lines {
		out = append(out, reindentLine(line, currentUnit, targetUnit))
	}
	return strings.Join(out, "\n")
}

// detectIndentUnit finds the smallest leading whitespace used in the file.
func detectIndentUnit(lines []string) string {
	minSpaces := 0
	hasTabs := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		leading := leadingWhitespace(line)
		if leading == "" {
			continue
		}
		if strings.HasPrefix(leading, "\t") {
			hasTabs = true
			break
		}
		n := len(leading)
		if minSpaces == 0 || n < minSpaces {
			minSpaces = n
		}
	}

	if hasTabs {
		return "\t"
	}
	if minSpaces > 0 {
		return strings.Repeat(" ", minSpaces)
	}
	return ""
}

// leadingWhitespace returns the leading whitespace characters of a line.
func leadingWhitespace(line string) string {
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			return line[:i]
		}
	}
	return line
}

// reindentLine replaces the leading indent unit occurrences in a line.
func reindentLine(line, from, to string) string {
	leading := leadingWhitespace(line)
	if leading == "" {
		return line
	}
	count := 0
	remaining := leading
	for strings.HasPrefix(remaining, from) {
		count++
		remaining = remaining[len(from):]
	}
	if count == 0 {
		return line
	}
	// Preserve any partial leading whitespace beyond counted units.
	newLeading := strings.Repeat(to, count) + remaining
	return newLeading + line[len(leading):]
}

// formatHTML performs basic heuristic HTML indentation:
// one tag per line, children indented relative to parent.
func formatHTML(code string, indentSize int, useTabs bool) string {
	var indentStr string
	if useTabs {
		indentStr = "\t"
	} else {
		indentStr = strings.Repeat(" ", indentSize)
	}

	// Self-closing and void elements that do not increase indent.
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true, "!doctype": true,
	}

	// Tokenize the HTML into a stream of tags and text nodes.
	tokens := tokenizeHTML(code)

	var buf strings.Builder
	depth := 0

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}

		if strings.HasPrefix(tok, "<!--") {
			// Comment — emit at current indent.
			buf.WriteString(strings.Repeat(indentStr, depth))
			buf.WriteString(tok)
			buf.WriteByte('\n')
			continue
		}

		if strings.HasPrefix(tok, "</") {
			// Closing tag — dedent first.
			tagName := extractTagName(tok)
			if !voidElements[strings.ToLower(tagName)] {
				if depth > 0 {
					depth--
				}
			}
			buf.WriteString(strings.Repeat(indentStr, depth))
			buf.WriteString(tok)
			buf.WriteByte('\n')
			continue
		}

		if strings.HasPrefix(tok, "<") {
			tagName := extractTagName(tok)
			lower := strings.ToLower(tagName)
			selfClose := strings.HasSuffix(tok, "/>") || voidElements[lower]

			buf.WriteString(strings.Repeat(indentStr, depth))
			buf.WriteString(tok)
			buf.WriteByte('\n')

			if !selfClose {
				depth++
			}
			continue
		}

		// Text node — emit at current indent if non-empty.
		if strings.TrimSpace(tok) != "" {
			buf.WriteString(strings.Repeat(indentStr, depth))
			buf.WriteString(strings.TrimSpace(tok))
			buf.WriteByte('\n')
		}
	}

	return strings.TrimRight(buf.String(), "\n")
}

// tokenizeHTML splits HTML into a flat list of tags and text segments.
func tokenizeHTML(html string) []string {
	var tokens []string
	for len(html) > 0 {
		if idx := strings.Index(html, "<"); idx == -1 {
			tokens = append(tokens, html)
			break
		} else {
			if idx > 0 {
				tokens = append(tokens, html[:idx])
			}
			html = html[idx:]
			end := strings.Index(html, ">")
			if end == -1 {
				tokens = append(tokens, html)
				break
			}
			tokens = append(tokens, html[:end+1])
			html = html[end+1:]
		}
	}
	return tokens
}

// extractTagName extracts the tag name from a tag token like "<div class='foo'>".
func extractTagName(tag string) string {
	s := tag
	if strings.HasPrefix(s, "</") {
		s = s[2:]
	} else if strings.HasPrefix(s, "<") {
		s = s[1:]
	}
	s = strings.TrimSpace(s)
	end := strings.IndexAny(s, " \t\n/>")
	if end == -1 {
		return strings.Trim(s, "<>/")
	}
	return s[:end]
}

// formatCSS normalizes CSS rule blocks: one property per line, consistent indentation.
func formatCSS(code string, indentSize int, useTabs bool) string {
	var indentStr string
	if useTabs {
		indentStr = "\t"
	} else {
		indentStr = strings.Repeat(" ", indentSize)
	}

	// Split by '{' and '}' while preserving them.
	var buf strings.Builder
	depth := 0
	i := 0

	for i < len(code) {
		ch := code[i]
		switch ch {
		case '{':
			// Flush selector (trimmed) + opening brace.
			buf.WriteString(" {\n")
			depth++
			i++
			// Skip whitespace after '{'
			for i < len(code) && (code[i] == ' ' || code[i] == '\t' || code[i] == '\n' || code[i] == '\r') {
				i++
			}

		case '}':
			depth--
			if depth < 0 {
				depth = 0
			}
			buf.WriteString(strings.Repeat(indentStr, depth))
			buf.WriteString("}\n")
			i++
			// Skip whitespace after '}'
			for i < len(code) && (code[i] == ' ' || code[i] == '\t' || code[i] == '\n' || code[i] == '\r') {
				i++
			}

		case ';':
			buf.WriteString(";\n")
			i++
			// Skip whitespace after ';'
			for i < len(code) && (code[i] == ' ' || code[i] == '\t' || code[i] == '\n' || code[i] == '\r') {
				i++
			}
			// Write indent for next property.
			if i < len(code) && code[i] != '}' {
				buf.WriteString(strings.Repeat(indentStr, depth))
			}

		case '\n', '\r', '\t':
			// Skip raw newlines/tabs outside properties; handled by our logic.
			i++

		case ' ':
			// Preserve spaces within declarations, but skip leading spaces after newline.
			if buf.Len() == 0 || buf.String()[buf.Len()-1] == '\n' {
				i++
			} else {
				buf.WriteByte(ch)
				i++
			}

		default:
			// Beginning of a selector or property value.
			// If we're at depth 0, this is a selector — emit indent (depth 0 → no indent).
			// If we're inside a rule (depth > 0), emit indent.
			if buf.Len() == 0 || buf.String()[buf.Len()-1] == '\n' {
				buf.WriteString(strings.Repeat(indentStr, depth))
			}
			buf.WriteByte(ch)
			i++
		}
	}

	result := strings.TrimRight(buf.String(), "\n")
	return result
}

// ─── code_metrics ─────────────────────────────────────────────────────────────

// MetricsInput is the input schema for the code_metrics tool.
type MetricsInput struct {
	Code     string `json:"code"`
	Language string `json:"language"` // go | typescript | python | generic
}

// MetricsOutput is the output schema for the code_metrics tool.
type MetricsOutput struct {
	LOC                int    `json:"loc"`
	SLOC               int    `json:"sloc"`
	BlankLines         int    `json:"blank_lines"`
	CommentLines       int    `json:"comment_lines"`
	Functions          int    `json:"functions"`
	ComplexityEstimate int    `json:"complexity_estimate"`
	Language           string `json:"language"`
}

// Metrics computes code metrics for the given source code.
func Metrics(_ context.Context, input MetricsInput) string {
	if strings.TrimSpace(input.Code) == "" {
		return errResult("code is required")
	}

	lang := strings.ToLower(strings.TrimSpace(input.Language))
	switch lang {
	case "go":
		return metricsGo(input.Code)
	case "typescript", "javascript", "ts", "js":
		return metricsRegex(input.Code, "typescript")
	case "python":
		return metricsRegex(input.Code, "python")
	case "generic", "":
		return metricsRegex(input.Code, "generic")
	default:
		return errResult(fmt.Sprintf("unsupported language %q (supported: go, typescript, python, generic)", input.Language))
	}
}

// metricsGo uses go/ast for accurate Go metrics.
func metricsGo(code string) string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", code, parser.ParseComments)

	lines := strings.Split(code, "\n")
	loc := len(lines)

	// Count blank and comment lines via raw text analysis.
	blank, commentLines := countBlankAndComments(lines, "go")
	sloc := loc - blank - commentLines

	if sloc < 0 {
		sloc = 0
	}

	var funcs int
	var complexity int

	if err == nil {
		// Count function declarations.
		ast.Inspect(f, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.FuncDecl:
				funcs++
			case *ast.FuncLit:
				funcs++
			}
			return true
		})

		// Count complexity: if, else, for, range, select, case, &&, ||.
		ast.Inspect(f, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.IfStmt:
				complexity++
			case *ast.ForStmt:
				complexity++
			case *ast.RangeStmt:
				complexity++
			case *ast.CaseClause:
				complexity++
			case *ast.CommClause:
				complexity++
			case *ast.SelectStmt:
				// select itself adds 1
				complexity++
			}
			return true
		})

		// Also count logical operators via token scan.
		complexity += countLogicalOps(code)
	} else {
		// Fallback: regex-based counting.
		funcs = countPattern(code, `\bfunc\b`)
		complexity = countComplexityRegex(code, "go")
	}

	return resultJSON(MetricsOutput{
		LOC:                loc,
		SLOC:               sloc,
		BlankLines:         blank,
		CommentLines:       commentLines,
		Functions:          funcs,
		ComplexityEstimate: complexity,
		Language:           "go",
	})
}

// metricsRegex uses regex heuristics for non-Go languages.
func metricsRegex(code, lang string) string {
	lines := strings.Split(code, "\n")
	loc := len(lines)
	blank, commentLines := countBlankAndComments(lines, lang)
	sloc := loc - blank - commentLines
	if sloc < 0 {
		sloc = 0
	}

	funcs := countFunctionsRegex(code, lang)
	complexity := countComplexityRegex(code, lang)

	return resultJSON(MetricsOutput{
		LOC:                loc,
		SLOC:               sloc,
		BlankLines:         blank,
		CommentLines:       commentLines,
		Functions:          funcs,
		ComplexityEstimate: complexity,
		Language:           lang,
	})
}

// countBlankAndComments counts blank lines and comment lines.
func countBlankAndComments(lines []string, lang string) (blank, comments int) {
	inBlockComment := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blank++
			continue
		}

		switch lang {
		case "go", "typescript", "javascript", "ts", "js", "generic":
			if inBlockComment {
				comments++
				if strings.Contains(trimmed, "*/") {
					inBlockComment = false
				}
				continue
			}
			if strings.HasPrefix(trimmed, "//") {
				comments++
				continue
			}
			if strings.HasPrefix(trimmed, "/*") {
				comments++
				if !strings.Contains(trimmed[2:], "*/") {
					inBlockComment = true
				}
				continue
			}
			// Inline block comment — not a comment-only line.
		case "python":
			if inBlockComment {
				comments++
				if strings.Contains(trimmed, `"""`) || strings.Contains(trimmed, `'''`) {
					// Crude: if the closing quotes appear, end block comment.
					inBlockComment = false
				}
				continue
			}
			if strings.HasPrefix(trimmed, "#") {
				comments++
				continue
			}
			// Detect docstring start.
			if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
				comments++
				// Check if multi-line docstring.
				rest := trimmed[3:]
				if !strings.Contains(rest, `"""`) && !strings.Contains(rest, `'''`) {
					inBlockComment = true
				}
				continue
			}
		}
	}
	return blank, comments
}

// countLogicalOps counts && and || tokens in Go source.
func countLogicalOps(code string) int {
	return strings.Count(code, "&&") + strings.Count(code, "||")
}

// countPattern counts non-overlapping regex matches.
func countPattern(code, pattern string) int {
	re := regexp.MustCompile(pattern)
	return len(re.FindAllString(code, -1))
}

// countFunctionsRegex counts function declarations via language-specific regex.
func countFunctionsRegex(code, lang string) int {
	switch lang {
	case "typescript", "javascript", "ts", "js":
		// function foo, const foo = function, const foo = (...) =>, foo: function
		re := regexp.MustCompile(`\bfunction\s+\w+|\bfunction\s*\(|=>\s*{|=\s*\(.*?\)\s*=>`)
		return len(re.FindAllString(code, -1))
	case "python":
		re := regexp.MustCompile(`(?m)^\s*def\s+\w+`)
		return len(re.FindAllString(code, -1))
	default:
		re := regexp.MustCompile(`\bfunction\b|\bfunc\b|\bdef\b`)
		return len(re.FindAllString(code, -1))
	}
}

// countComplexityRegex counts branching points via regex.
func countComplexityRegex(code, lang string) int {
	count := 0
	switch lang {
	case "python":
		patterns := []string{
			`\bif\b`, `\belif\b`, `\bfor\b`, `\bwhile\b`,
			`\bcase\b`, `\band\b`, `\bor\b`,
		}
		for _, p := range patterns {
			count += countPattern(code, p)
		}
	case "go":
		patterns := []string{
			`\bif\b`, `\bfor\b`, `\bcase\b`, `\bselect\b`,
		}
		for _, p := range patterns {
			count += countPattern(code, p)
		}
		count += strings.Count(code, "&&") + strings.Count(code, "||")
	default: // typescript, javascript, generic
		patterns := []string{
			`\bif\b`, `\belse\s+if\b`, `\bfor\b`, `\bwhile\b`,
			`\bcase\b`, `\?\s`, // ternary
		}
		for _, p := range patterns {
			count += countPattern(code, p)
		}
		count += strings.Count(code, "&&") + strings.Count(code, "||")
	}
	return count
}

// ─── code_template ────────────────────────────────────────────────────────────

// TemplateInput is the input schema for the code_template tool.
type TemplateInput struct {
	Template string `json:"template"`
	Context  string `json:"context"` // JSON object with variable bindings
	Engine   string `json:"engine"`  // go | mustache (default: go)
}

// TemplateOutput is the output schema for the code_template tool.
type TemplateOutput struct {
	Result string `json:"result"`
}

// Template renders a template with the given context.
func Template(_ context.Context, input TemplateInput) string {
	if strings.TrimSpace(input.Template) == "" {
		return errResult("template is required")
	}
	if strings.TrimSpace(input.Context) == "" {
		return errResult("context is required")
	}

	engine := strings.ToLower(strings.TrimSpace(input.Engine))
	if engine == "" {
		engine = "go"
	}

	// Parse context JSON.
	var ctx map[string]any
	if err := json.Unmarshal([]byte(input.Context), &ctx); err != nil {
		return errResult("invalid context JSON: " + err.Error())
	}

	switch engine {
	case "go":
		result, err := renderGoTemplate(input.Template, ctx)
		if err != nil {
			return errResult("template execution error: " + err.Error())
		}
		return resultJSON(TemplateOutput{Result: result})

	case "mustache":
		result, err := renderMustache(input.Template, ctx)
		if err != nil {
			return errResult("mustache error: " + err.Error())
		}
		return resultJSON(TemplateOutput{Result: result})

	default:
		return errResult(fmt.Sprintf("unsupported engine %q (supported: go, mustache)", input.Engine))
	}
}

// renderGoTemplate renders a text/template template with the given data.
func renderGoTemplate(tmplStr string, data map[string]any) (string, error) {
	tmpl, err := template.New("t").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute error: %w", err)
	}
	return buf.String(), nil
}

// ─── Mustache interpreter ────────────────────────────────────────────────────

// renderMustache implements a minimal Mustache renderer.
// Supported tags:
//   - {{variable}}           — variable interpolation (HTML-escaped not applied; caller controls escaping)
//   - {{{variable}}}         — unescaped variable (same as {{variable}} here since we use text/template semantics)
//   - {{#section}}...{{/section}} — truthy check + iteration over arrays
//   - {{^inverted}}...{{/inverted}} — inverted section (falsy / empty)
//   - {{! comment }}         — comment (rendered as empty string)
func renderMustache(tmpl string, data map[string]any) (string, error) {
	var buf strings.Builder
	err := mustacheRender(&buf, tmpl, data)
	return buf.String(), err
}

// mustacheRender recursively renders a Mustache template.
func mustacheRender(buf *strings.Builder, tmpl string, data map[string]any) error {
	for len(tmpl) > 0 {
		// Find next opening {{
		open := strings.Index(tmpl, "{{")
		if open == -1 {
			buf.WriteString(tmpl)
			break
		}

		// Write literal text before the tag.
		buf.WriteString(tmpl[:open])
		tmpl = tmpl[open:]

		// Determine tag kind.
		if strings.HasPrefix(tmpl, "{{{") {
			// Triple mustache — unescaped variable.
			close := strings.Index(tmpl, "}}}")
			if close == -1 {
				return fmt.Errorf("unclosed {{{")
			}
			key := strings.TrimSpace(tmpl[3:close])
			tmpl = tmpl[close+3:]
			buf.WriteString(mustacheValue(data, key))
			continue
		}

		close := strings.Index(tmpl, "}}")
		if close == -1 {
			return fmt.Errorf("unclosed {{")
		}

		inner := strings.TrimSpace(tmpl[2:close])
		tmpl = tmpl[close+2:]

		if len(inner) == 0 {
			continue
		}

		switch inner[0] {
		case '!':
			// Comment — discard.
			continue

		case '#':
			// Section start.
			key := strings.TrimSpace(inner[1:])
			body, rest, err := mustacheFindSection(tmpl, key, false)
			if err != nil {
				return err
			}
			tmpl = rest

			val := mustacheLookup(data, key)
			if mustacheTruthy(val) {
				// If it's a slice, iterate; otherwise render once with same data.
				if arr, ok := toSlice(val); ok {
					for _, item := range arr {
						child := mustacheMergeContext(data, key, item)
						if err := mustacheRender(buf, body, child); err != nil {
							return err
						}
					}
				} else {
					// Single truthy value — render once; if it's a map, push scope.
					var child map[string]any
					if m, ok := val.(map[string]any); ok {
						child = mustacheMergeMap(data, m)
					} else {
						child = data
					}
					if err := mustacheRender(buf, body, child); err != nil {
						return err
					}
				}
			}

		case '^':
			// Inverted section.
			key := strings.TrimSpace(inner[1:])
			body, rest, err := mustacheFindSection(tmpl, key, true)
			if err != nil {
				return err
			}
			tmpl = rest

			val := mustacheLookup(data, key)
			if !mustacheTruthy(val) {
				if err := mustacheRender(buf, body, data); err != nil {
					return err
				}
			}

		case '/':
			// Closing tag found outside section — error.
			return fmt.Errorf("unexpected closing tag {{/%s}}", strings.TrimSpace(inner[1:]))

		default:
			// Variable interpolation.
			buf.WriteString(mustacheValue(data, inner))
		}
	}
	return nil
}

// mustacheFindSection finds the body between {{#key}}...{{/key}} or {{^key}}...{{/key}}.
// Returns (body, remaining_template, error).
func mustacheFindSection(tmpl, key string, inverted bool) (string, string, error) {
	prefix := "#"
	if inverted {
		prefix = "^"
	}
	closeTag := "{{/" + key + "}}"

	// We need to handle nested sections with the same key.
	depth := 1
	pos := 0
	for pos < len(tmpl) {
		open := strings.Index(tmpl[pos:], "{{")
		if open == -1 {
			return "", "", fmt.Errorf("unclosed section {{%s%s}}", prefix, key)
		}
		abs := pos + open

		close := strings.Index(tmpl[abs:], "}}")
		if close == -1 {
			return "", "", fmt.Errorf("malformed template: unclosed {{")
		}
		closeAbs := abs + close + 2
		inner := strings.TrimSpace(tmpl[abs+2 : abs+close])

		if inner == "#"+key || inner == "^"+key {
			depth++
			pos = closeAbs
		} else if inner == "/"+key {
			depth--
			if depth == 0 {
				// Found the matching closing tag.
				body := tmpl[:abs]
				rest := tmpl[closeAbs:]
				return body, rest, nil
			}
			pos = closeAbs
		} else {
			pos = closeAbs
		}
	}
	_ = closeTag
	return "", "", fmt.Errorf("unclosed section {{%s%s}}", prefix, key)
}

// mustacheLookup looks up a key in the data map, supporting dot notation.
func mustacheLookup(data map[string]any, key string) any {
	if key == "." {
		return data
	}
	parts := strings.SplitN(key, ".", 2)
	val, ok := data[parts[0]]
	if !ok {
		return nil
	}
	if len(parts) == 1 {
		return val
	}
	// Dot notation: recurse.
	if sub, ok := val.(map[string]any); ok {
		return mustacheLookup(sub, parts[1])
	}
	return nil
}

// mustacheValue renders a variable key as a string.
func mustacheValue(data map[string]any, key string) string {
	val := mustacheLookup(data, key)
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		// Format integers without decimals.
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// mustacheTruthy returns true if a value is considered truthy in Mustache.
func mustacheTruthy(val any) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	}
	return true
}

// toSlice attempts to convert a value to []any.
func toSlice(val any) ([]any, bool) {
	if arr, ok := val.([]any); ok {
		return arr, true
	}
	return nil, false
}

// mustacheMergeContext creates a new context with the section item merged in.
func mustacheMergeContext(parent map[string]any, key string, item any) map[string]any {
	child := make(map[string]any, len(parent)+2)
	for k, v := range parent {
		child[k] = v
	}
	child[key] = item
	// Also expose the item's own fields if it's a map.
	if m, ok := item.(map[string]any); ok {
		for k, v := range m {
			child[k] = v
		}
	}
	return child
}

// mustacheMergeMap creates a new context merging a sub-map into parent.
func mustacheMergeMap(parent, sub map[string]any) map[string]any {
	child := make(map[string]any, len(parent)+len(sub))
	for k, v := range parent {
		child[k] = v
	}
	for k, v := range sub {
		child[k] = v
	}
	return child
}

// ─── unicode helpers ──────────────────────────────────────────────────────────

// isLetter reports whether a rune is a letter (used for tag name parsing).
func isLetter(r rune) bool {
	return unicode.IsLetter(r)
}

// ensure isLetter is used (suppress unused import warnings)
var _ = isLetter
