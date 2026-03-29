// Package datafmt implements MCP tools for data formatting and transformation.
// Tools: data_json_format, data_yaml_convert, data_csv_convert, data_jsonpath,
// data_schema_validate, data_diff.
package datafmt

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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

// ─── data_json_format ────────────────────────────────────────────────────────

// FormatJSONInput is the input schema for the data_json_format tool.
type FormatJSONInput struct {
	JSON   string `json:"json"`
	Indent string `json:"indent"`
}

// FormatJSON pretty-prints a JSON string using the given indent string.
// On parse failure it returns {"error":"...", "line":N, "column":N}.
func FormatJSON(_ context.Context, input FormatJSONInput) string {
	if strings.TrimSpace(input.JSON) == "" {
		return errResult("json is required")
	}
	indent := input.Indent
	if indent == "" {
		indent = "  "
	}

	var v any
	if err := json.Unmarshal([]byte(input.JSON), &v); err != nil {
		if se, ok := err.(*json.SyntaxError); ok {
			line, col := offsetToLineCol(input.JSON, int(se.Offset))
			b, _ := json.Marshal(map[string]any{
				"error":  se.Error(),
				"line":   line,
				"column": col,
			})
			return string(b)
		}
		return errResult("invalid JSON: " + err.Error())
	}

	out, err := json.MarshalIndent(v, "", indent)
	if err != nil {
		return errResult("marshal failed: " + err.Error())
	}
	return resultJSON(map[string]string{"result": string(out)})
}

// offsetToLineCol converts a byte offset into 1-based line/column numbers.
func offsetToLineCol(s string, offset int) (line, col int) {
	if offset > len(s) {
		offset = len(s)
	}
	line = 1
	lastNewline := 0
	for i := 0; i < offset; i++ {
		if s[i] == '\n' {
			line++
			lastNewline = i + 1
		}
	}
	col = offset - lastNewline + 1
	return
}

// ─── data_yaml_convert ───────────────────────────────────────────────────────

// YAMLConvertInput is the input schema for the data_yaml_convert tool.
type YAMLConvertInput struct {
	Input string `json:"input"`
	From  string `json:"from"` // json | yaml
	To    string `json:"to"`   // json | yaml
}

// YAMLConvert converts between JSON and YAML formats.
func YAMLConvert(_ context.Context, input YAMLConvertInput) string {
	if strings.TrimSpace(input.Input) == "" {
		return errResult("input is required")
	}
	if input.From == input.To {
		return resultJSON(map[string]string{"result": input.Input})
	}

	switch input.From + "->" + input.To {
	case "json->yaml":
		var v any
		if err := json.Unmarshal([]byte(input.Input), &v); err != nil {
			return errResult("invalid JSON: " + err.Error())
		}
		// Convert map[string]any to map[string]any — yaml.v3 requires this.
		v = normalizeForYAML(v)
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(v); err != nil {
			return errResult("YAML encode failed: " + err.Error())
		}
		enc.Close()
		return resultJSON(map[string]string{"result": strings.TrimRight(buf.String(), "\n")})

	case "yaml->json":
		var v any
		if err := yaml.Unmarshal([]byte(input.Input), &v); err != nil {
			return errResult("invalid YAML: " + err.Error())
		}
		v = normalizeFromYAML(v)
		out, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return errResult("JSON encode failed: " + err.Error())
		}
		return resultJSON(map[string]string{"result": string(out)})

	default:
		return errResult(fmt.Sprintf("unsupported conversion: %s->%s (supported: json|yaml)", input.From, input.To))
	}
}

// normalizeForYAML recursively converts map[interface{}]interface{} to map[string]any.
func normalizeForYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = normalizeForYAML(vv)
		}
		return out
	case []any:
		for i, vv := range val {
			val[i] = normalizeForYAML(vv)
		}
		return val
	default:
		return v
	}
}

// normalizeFromYAML recursively converts map[string]any and map[interface{}]interface{}
// (produced by yaml.v3) into forms that encoding/json can marshal.
func normalizeFromYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[k] = normalizeFromYAML(vv)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[fmt.Sprintf("%v", k)] = normalizeFromYAML(vv)
		}
		return out
	case []any:
		for i, vv := range val {
			val[i] = normalizeFromYAML(vv)
		}
		return val
	default:
		return v
	}
}

// ─── data_csv_convert ────────────────────────────────────────────────────────

// CSVConvertInput is the input schema for the data_csv_convert tool.
type CSVConvertInput struct {
	Input     string `json:"input"`
	From      string `json:"from"`      // csv | json
	To        string `json:"to"`        // csv | json
	Separator string `json:"separator"` // default ","
	HasHeader bool   `json:"has_header"`
}

// CSVConvert converts between CSV and JSON formats.
func CSVConvert(_ context.Context, input CSVConvertInput) string {
	if strings.TrimSpace(input.Input) == "" {
		return errResult("input is required")
	}
	sep := ','
	if input.Separator != "" {
		runes := []rune(input.Separator)
		if len(runes) != 1 {
			return errResult("separator must be a single character")
		}
		sep = runes[0]
	}
	hasHeader := input.HasHeader // caller sets it; default handled by MCP layer

	switch input.From + "->" + input.To {
	case "csv->json":
		result, err := csvToJSON(input.Input, sep, hasHeader)
		if err != nil {
			return errResult(err.Error())
		}
		return resultJSON(map[string]string{"result": result})

	case "json->csv":
		result, err := jsonToCSV(input.Input, sep)
		if err != nil {
			return errResult(err.Error())
		}
		return resultJSON(map[string]string{"result": result})

	default:
		return errResult(fmt.Sprintf("unsupported conversion: %s->%s (supported: csv|json)", input.From, input.To))
	}
}

func csvToJSON(raw string, sep rune, hasHeader bool) (string, error) {
	r := csv.NewReader(strings.NewReader(raw))
	r.Comma = sep
	r.TrimLeadingSpace = true

	records, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("CSV parse error: %w", err)
	}
	if len(records) == 0 {
		out, _ := json.Marshal([]any{})
		return string(out), nil
	}

	var result []any

	if hasHeader {
		headers := records[0]
		for _, row := range records[1:] {
			obj := make(map[string]any, len(headers))
			for i, h := range headers {
				if i < len(row) {
					obj[h] = row[i]
				} else {
					obj[h] = ""
				}
			}
			result = append(result, obj)
		}
	} else {
		for _, row := range records {
			obj := make(map[string]any, len(row))
			for i, v := range row {
				obj[strconv.Itoa(i)] = v
			}
			result = append(result, obj)
		}
	}

	if result == nil {
		result = []any{}
	}
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON marshal error: %w", err)
	}
	return string(out), nil
}

func jsonToCSV(raw string, sep rune) (string, error) {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return "", fmt.Errorf("invalid JSON array of objects: %w", err)
	}
	if len(rows) == 0 {
		return "", nil
	}

	// Collect ordered headers from first object.
	headers := make([]string, 0)
	seen := make(map[string]bool)
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	// Sort headers deterministically.
	sortStrings(headers)

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	w.Comma = sep

	if err := w.Write(headers); err != nil {
		return "", err
	}
	for _, row := range rows {
		rec := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := row[h]; ok {
				rec[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := w.Write(rec); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// sortStrings is a simple insertion sort for string slices (avoids importing sort).
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}

// ─── data_jsonpath ───────────────────────────────────────────────────────────

// JSONPathInput is the input schema for the data_jsonpath tool.
type JSONPathInput struct {
	JSON string `json:"json"`
	Path string `json:"path"`
}

// JSONPath evaluates a minimal JSONPath expression against a JSON document.
// Supported: $, .field, [N], .*, [*]
func JSONPath(_ context.Context, input JSONPathInput) string {
	if strings.TrimSpace(input.JSON) == "" {
		return errResult("json is required")
	}
	if strings.TrimSpace(input.Path) == "" {
		return errResult("path is required")
	}

	var root any
	if err := json.Unmarshal([]byte(input.JSON), &root); err != nil {
		return errResult("invalid JSON: " + err.Error())
	}

	results, err := evalJSONPath(root, input.Path)
	if err != nil {
		return errResult(err.Error())
	}

	// If a single result, unwrap it.
	var val any
	if len(results) == 1 {
		val = results[0]
	} else {
		val = results
	}

	b, err := json.Marshal(map[string]any{"result": val})
	if err != nil {
		return errResult("marshal failed: " + err.Error())
	}
	return string(b)
}

// evalJSONPath evaluates a JSONPath expression and returns matching values.
func evalJSONPath(root any, path string) ([]any, error) {
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("JSONPath must start with '$'")
	}
	tokens, err := tokenizeJSONPath(path[1:]) // strip leading '$'
	if err != nil {
		return nil, err
	}
	results := []any{root}
	for _, tok := range tokens {
		var next []any
		for _, cur := range results {
			vals, err := applyToken(cur, tok)
			if err != nil {
				return nil, err
			}
			next = append(next, vals...)
		}
		results = next
	}
	return results, nil
}

// jsonPathToken represents a single step in a JSONPath expression.
type jsonPathToken struct {
	kind  string // "field", "index", "wildcard"
	field string
	index int
}

func tokenizeJSONPath(path string) ([]jsonPathToken, error) {
	var tokens []jsonPathToken
	i := 0
	for i < len(path) {
		switch path[i] {
		case '.':
			i++
			if i >= len(path) {
				return nil, fmt.Errorf("unexpected end of path after '.'")
			}
			if path[i] == '*' {
				tokens = append(tokens, jsonPathToken{kind: "wildcard"})
				i++
			} else {
				// Read field name until '.', '[', or end.
				j := i
				for j < len(path) && path[j] != '.' && path[j] != '[' {
					j++
				}
				tokens = append(tokens, jsonPathToken{kind: "field", field: path[i:j]})
				i = j
			}
		case '[':
			i++
			j := i
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j >= len(path) {
				return nil, fmt.Errorf("unclosed '[' in path")
			}
			inner := path[i:j]
			i = j + 1 // skip ']'
			if inner == "*" {
				tokens = append(tokens, jsonPathToken{kind: "wildcard"})
			} else {
				idx, err := strconv.Atoi(inner)
				if err != nil {
					return nil, fmt.Errorf("invalid array index %q", inner)
				}
				tokens = append(tokens, jsonPathToken{kind: "index", index: idx})
			}
		default:
			return nil, fmt.Errorf("unexpected character %q in path", string(path[i]))
		}
	}
	return tokens, nil
}

func applyToken(cur any, tok jsonPathToken) ([]any, error) {
	switch tok.kind {
	case "field":
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, nil // field on non-object → no match
		}
		v, exists := m[tok.field]
		if !exists {
			return nil, nil
		}
		return []any{v}, nil

	case "index":
		arr, ok := cur.([]any)
		if !ok {
			return nil, nil
		}
		idx := tok.index
		if idx < 0 {
			idx = len(arr) + idx
		}
		if idx < 0 || idx >= len(arr) {
			return nil, fmt.Errorf("array index %d out of bounds (len=%d)", tok.index, len(arr))
		}
		return []any{arr[idx]}, nil

	case "wildcard":
		switch v := cur.(type) {
		case map[string]any:
			out := make([]any, 0, len(v))
			for _, val := range v {
				out = append(out, val)
			}
			return out, nil
		case []any:
			return v, nil
		default:
			return nil, nil
		}
	}
	return nil, fmt.Errorf("unknown token kind %q", tok.kind)
}

// ─── data_schema_validate ────────────────────────────────────────────────────

// SchemaValidateInput is the input schema for the data_schema_validate tool.
type SchemaValidateInput struct {
	JSON   string `json:"json"`
	Schema string `json:"schema"`
}

// SchemaValidate validates a JSON document against a JSON Schema (basic subset).
func SchemaValidate(_ context.Context, input SchemaValidateInput) string {
	if strings.TrimSpace(input.JSON) == "" {
		return errResult("json is required")
	}
	if strings.TrimSpace(input.Schema) == "" {
		return errResult("schema is required")
	}

	var doc any
	if err := json.Unmarshal([]byte(input.JSON), &doc); err != nil {
		return errResult("invalid JSON document: " + err.Error())
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(input.Schema), &schema); err != nil {
		return errResult("invalid JSON schema: " + err.Error())
	}

	errs := validateSchema(doc, schema, "$")
	if len(errs) == 0 {
		return resultJSON(map[string]any{"valid": true})
	}
	return resultJSON(map[string]any{"valid": false, "errors": errs})
}

// validateSchema validates doc against schema at path, returning error descriptions.
func validateSchema(doc any, schema map[string]any, path string) []string {
	var errs []string

	// type check
	if typeVal, ok := schema["type"]; ok {
		typeStr, _ := typeVal.(string)
		if !matchesType(doc, typeStr) {
			errs = append(errs, fmt.Sprintf("%s: expected type %q, got %s", path, typeStr, jsonTypeName(doc)))
		}
	}

	// enum check
	if enumVal, ok := schema["enum"]; ok {
		if arr, ok := enumVal.([]any); ok {
			if !inEnum(doc, arr) {
				errs = append(errs, fmt.Sprintf("%s: value not in enum", path))
			}
		}
	}

	// string validations
	if str, ok := doc.(string); ok {
		if minLen, ok := numericField(schema, "minLength"); ok {
			if float64(len(str)) < minLen {
				errs = append(errs, fmt.Sprintf("%s: string length %d is less than minLength %g", path, len(str), minLen))
			}
		}
		if maxLen, ok := numericField(schema, "maxLength"); ok {
			if float64(len(str)) > maxLen {
				errs = append(errs, fmt.Sprintf("%s: string length %d exceeds maxLength %g", path, len(str), maxLen))
			}
		}
		if patternVal, ok := schema["pattern"]; ok {
			if patStr, ok := patternVal.(string); ok {
				re, err := regexp.Compile(patStr)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s: invalid pattern %q: %v", path, patStr, err))
				} else if !re.MatchString(str) {
					errs = append(errs, fmt.Sprintf("%s: string does not match pattern %q", path, patStr))
				}
			}
		}
	}

	// number validations
	if num, ok := toFloat64(doc); ok {
		if min, ok := numericField(schema, "minimum"); ok {
			if num < min {
				errs = append(errs, fmt.Sprintf("%s: value %g is less than minimum %g", path, num, min))
			}
		}
		if max, ok := numericField(schema, "maximum"); ok {
			if num > max {
				errs = append(errs, fmt.Sprintf("%s: value %g exceeds maximum %g", path, num, max))
			}
		}
		if excMin, ok := numericField(schema, "exclusiveMinimum"); ok {
			if num <= excMin {
				errs = append(errs, fmt.Sprintf("%s: value %g must be greater than exclusiveMinimum %g", path, num, excMin))
			}
		}
		if excMax, ok := numericField(schema, "exclusiveMaximum"); ok {
			if num >= excMax {
				errs = append(errs, fmt.Sprintf("%s: value %g must be less than exclusiveMaximum %g", path, num, excMax))
			}
		}
	}

	// object validations
	if obj, ok := doc.(map[string]any); ok {
		// required
		if reqVal, ok := schema["required"]; ok {
			if reqArr, ok := reqVal.([]any); ok {
				for _, req := range reqArr {
					if reqStr, ok := req.(string); ok {
						if _, exists := obj[reqStr]; !exists {
							errs = append(errs, fmt.Sprintf("%s: missing required property %q", path, reqStr))
						}
					}
				}
			}
		}
		// properties
		if propsVal, ok := schema["properties"]; ok {
			if props, ok := propsVal.(map[string]any); ok {
				for propName, propSchema := range props {
					if propSchemaMap, ok := propSchema.(map[string]any); ok {
						if val, exists := obj[propName]; exists {
							childErrs := validateSchema(val, propSchemaMap, path+"."+propName)
							errs = append(errs, childErrs...)
						}
					}
				}
			}
		}
		// additionalProperties
		if addPropsVal, ok := schema["additionalProperties"]; ok {
			if addPropsAllowed, ok := addPropsVal.(bool); ok && !addPropsAllowed {
				if propsVal, ok := schema["properties"]; ok {
					if props, ok := propsVal.(map[string]any); ok {
						for key := range obj {
							if _, defined := props[key]; !defined {
								errs = append(errs, fmt.Sprintf("%s: additional property %q not allowed", path, key))
							}
						}
					}
				}
			}
		}
	}

	// array validations
	if arr, ok := doc.([]any); ok {
		if minItems, ok := numericField(schema, "minItems"); ok {
			if float64(len(arr)) < minItems {
				errs = append(errs, fmt.Sprintf("%s: array length %d is less than minItems %g", path, len(arr), minItems))
			}
		}
		if maxItems, ok := numericField(schema, "maxItems"); ok {
			if float64(len(arr)) > maxItems {
				errs = append(errs, fmt.Sprintf("%s: array length %d exceeds maxItems %g", path, len(arr), maxItems))
			}
		}
		if itemsVal, ok := schema["items"]; ok {
			if itemSchema, ok := itemsVal.(map[string]any); ok {
				for i, item := range arr {
					childErrs := validateSchema(item, itemSchema, fmt.Sprintf("%s[%d]", path, i))
					errs = append(errs, childErrs...)
				}
			}
		}
	}

	return errs
}

func matchesType(v any, typeName string) bool {
	switch typeName {
	case "string":
		_, ok := v.(string)
		return ok
	case "number":
		switch v.(type) {
		case float64, float32, int, int64:
			return true
		}
		return false
	case "integer":
		if f, ok := v.(float64); ok {
			return f == math.Trunc(f)
		}
		switch v.(type) {
		case int, int64:
			return true
		}
		return false
	case "boolean":
		_, ok := v.(bool)
		return ok
	case "null":
		return v == nil
	case "array":
		_, ok := v.([]any)
		return ok
	case "object":
		_, ok := v.(map[string]any)
		return ok
	}
	return true // unknown type — don't fail
}

func jsonTypeName(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case bool:
		return "boolean"
	case float64, float32, int, int64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
}

func inEnum(v any, arr []any) bool {
	vb, _ := json.Marshal(v)
	for _, item := range arr {
		ib, _ := json.Marshal(item)
		if bytes.Equal(vb, ib) {
			return true
		}
	}
	return false
}

func numericField(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	return toFloat64(v)
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// ─── data_diff ───────────────────────────────────────────────────────────────

// DiffInput is the input schema for the data_diff tool.
type DiffInput struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Format string `json:"format"` // json | yaml, default json
}

// DiffEntry represents a changed key in the diff.
type DiffEntry struct {
	Key  string `json:"key"`
	From any    `json:"from"`
	To   any    `json:"to"`
}

// DiffOutput is the output for the data_diff tool.
type DiffOutput struct {
	Added   []string    `json:"added"`
	Removed []string    `json:"removed"`
	Changed []DiffEntry `json:"changed"`
}

// Diff performs a structural diff between two JSON or YAML documents.
func Diff(_ context.Context, input DiffInput) string {
	if strings.TrimSpace(input.A) == "" {
		return errResult("a is required")
	}
	if strings.TrimSpace(input.B) == "" {
		return errResult("b is required")
	}
	format := input.Format
	if format == "" {
		format = "json"
	}

	mapA, err := parseToMap(input.A, format)
	if err != nil {
		return errResult("could not parse 'a': " + err.Error())
	}
	mapB, err := parseToMap(input.B, format)
	if err != nil {
		return errResult("could not parse 'b': " + err.Error())
	}

	out := DiffOutput{
		Added:   []string{},
		Removed: []string{},
		Changed: []DiffEntry{},
	}

	// Find added and changed keys.
	for k, vb := range mapB {
		if va, exists := mapA[k]; !exists {
			out.Added = append(out.Added, k)
		} else {
			if !deepEqual(va, vb) {
				out.Changed = append(out.Changed, DiffEntry{Key: k, From: va, To: vb})
			}
		}
	}

	// Find removed keys.
	for k := range mapA {
		if _, exists := mapB[k]; !exists {
			out.Removed = append(out.Removed, k)
		}
	}

	// Sort for deterministic output.
	sortStrings(out.Added)
	sortStrings(out.Removed)
	sortDiffEntries(out.Changed)

	return resultJSON(out)
}

func parseToMap(s, format string) (map[string]any, error) {
	var v any
	var err error

	switch format {
	case "yaml":
		err = yaml.Unmarshal([]byte(s), &v)
	default:
		err = json.Unmarshal([]byte(s), &v)
	}
	if err != nil {
		return nil, err
	}

	v = normalizeFromYAML(v)

	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input must be a JSON/YAML object (map), got %T", v)
	}
	return m, nil
}

// deepEqual compares two values using JSON serialization for semantic equality.
func deepEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return bytes.Equal(ab, bb)
}

func sortDiffEntries(s []DiffEntry) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j].Key > key.Key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
