package datafmt_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/datafmt"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func mustUnmarshal(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("unmarshal failed: %v — got: %s", err, s)
	}
	return m
}

func assertError(t *testing.T, result string) {
	t.Helper()
	m := mustUnmarshal(t, result)
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' key, got: %s", result)
	}
}

func assertResultString(t *testing.T, result string) string {
	t.Helper()
	m := mustUnmarshal(t, result)
	v, ok := m["result"]
	if !ok {
		t.Fatalf("expected 'result' key, got: %s", result)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("expected 'result' to be a string, got: %T", v)
	}
	return s
}

// ─── data_json_format ─────────────────────────────────────────────────────────

func TestFormatJSON(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     datafmt.FormatJSONInput
		wantError bool
		checkFn   func(t *testing.T, result string)
	}{
		{
			name: "happy path — compact json is pretty-printed",
			input: datafmt.FormatJSONInput{
				JSON:   `{"b":2,"a":1}`,
				Indent: "  ",
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				if !strings.Contains(s, "\n") {
					t.Errorf("expected pretty-printed output with newlines, got: %s", s)
				}
				// Verify it's valid JSON
				var v any
				if err := json.Unmarshal([]byte(s), &v); err != nil {
					t.Errorf("output is not valid JSON: %v", err)
				}
			},
		},
		{
			name: "happy path — custom indent (tab)",
			input: datafmt.FormatJSONInput{
				JSON:   `{"x":1}`,
				Indent: "\t",
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				if !strings.Contains(s, "\t") {
					t.Errorf("expected tab-indented output, got: %s", s)
				}
			},
		},
		{
			name: "error path — invalid JSON returns error with line/column",
			input: datafmt.FormatJSONInput{
				JSON:   `{"key": "value"`,
				Indent: "  ",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if _, ok := m["error"]; !ok {
					t.Errorf("expected 'error' key, got: %s", result)
				}
				if _, ok := m["line"]; !ok {
					t.Errorf("expected 'line' key for syntax error, got: %s", result)
				}
				if _, ok := m["column"]; !ok {
					t.Errorf("expected 'column' key for syntax error, got: %s", result)
				}
			},
		},
		{
			name:    "error path — empty json returns error",
			input:   datafmt.FormatJSONInput{JSON: "", Indent: "  "},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name: "default indent when empty",
			input: datafmt.FormatJSONInput{
				JSON:   `{"a":1}`,
				Indent: "",
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				if !strings.Contains(s, "  ") {
					t.Errorf("expected default 2-space indent, got: %s", s)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := datafmt.FormatJSON(ctx, tc.input)
			tc.checkFn(t, result)
		})
	}
}

// ─── data_yaml_convert ────────────────────────────────────────────────────────

func TestYAMLConvert(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   datafmt.YAMLConvertInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "happy path — JSON to YAML",
			input: datafmt.YAMLConvertInput{
				Input: `{"name":"alice","age":30}`,
				From:  "json",
				To:    "yaml",
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				if !strings.Contains(s, "name:") {
					t.Errorf("expected YAML with 'name:', got: %s", s)
				}
				if !strings.Contains(s, "alice") {
					t.Errorf("expected value 'alice' in YAML, got: %s", s)
				}
			},
		},
		{
			name: "happy path — YAML to JSON",
			input: datafmt.YAMLConvertInput{
				Input: "name: alice\nage: 30\n",
				From:  "yaml",
				To:    "json",
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				var m map[string]any
				if err := json.Unmarshal([]byte(s), &m); err != nil {
					t.Fatalf("output is not valid JSON: %v — got: %s", err, s)
				}
				if m["name"] != "alice" {
					t.Errorf("expected name=alice, got: %v", m["name"])
				}
			},
		},
		{
			name: "happy path — same format returns input",
			input: datafmt.YAMLConvertInput{
				Input: `{"a":1}`,
				From:  "json",
				To:    "json",
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				if s != `{"a":1}` {
					t.Errorf("expected unchanged input, got: %s", s)
				}
			},
		},
		{
			name: "error path — invalid JSON",
			input: datafmt.YAMLConvertInput{
				Input: `{bad json`,
				From:  "json",
				To:    "yaml",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name: "error path — unsupported format",
			input: datafmt.YAMLConvertInput{
				Input: `{"a":1}`,
				From:  "json",
				To:    "toml",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name:    "error path — empty input",
			input:   datafmt.YAMLConvertInput{Input: "", From: "json", To: "yaml"},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := datafmt.YAMLConvert(ctx, tc.input)
			tc.checkFn(t, result)
		})
	}
}

// ─── data_csv_convert ─────────────────────────────────────────────────────────

func TestCSVConvert(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   datafmt.CSVConvertInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "happy path — CSV with header to JSON",
			input: datafmt.CSVConvertInput{
				Input:     "name,age\nalice,30\nbob,25",
				From:      "csv",
				To:        "json",
				Separator: ",",
				HasHeader: true,
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				var rows []map[string]any
				if err := json.Unmarshal([]byte(s), &rows); err != nil {
					t.Fatalf("output is not valid JSON array: %v — got: %s", err, s)
				}
				if len(rows) != 2 {
					t.Errorf("expected 2 rows, got %d", len(rows))
				}
				if rows[0]["name"] != "alice" {
					t.Errorf("expected name=alice, got: %v", rows[0]["name"])
				}
			},
		},
		{
			name: "happy path — JSON to CSV",
			input: datafmt.CSVConvertInput{
				Input:     `[{"name":"alice","age":"30"},{"name":"bob","age":"25"}]`,
				From:      "json",
				To:        "csv",
				Separator: ",",
				HasHeader: true,
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				if !strings.Contains(s, "name") || !strings.Contains(s, "age") {
					t.Errorf("expected CSV headers, got: %s", s)
				}
				if !strings.Contains(s, "alice") {
					t.Errorf("expected 'alice' in CSV, got: %s", s)
				}
			},
		},
		{
			name: "happy path — CSV without header to JSON uses index keys",
			input: datafmt.CSVConvertInput{
				Input:     "alice,30\nbob,25",
				From:      "csv",
				To:        "json",
				Separator: ",",
				HasHeader: false,
			},
			checkFn: func(t *testing.T, result string) {
				s := assertResultString(t, result)
				var rows []map[string]any
				if err := json.Unmarshal([]byte(s), &rows); err != nil {
					t.Fatalf("output is not valid JSON array: %v", err)
				}
				if len(rows) != 2 {
					t.Errorf("expected 2 rows, got %d", len(rows))
				}
				// Keys should be "0", "1"
				if _, ok := rows[0]["0"]; !ok {
					t.Errorf("expected index-based key '0', got: %v", rows[0])
				}
			},
		},
		{
			name:    "error path — empty input",
			input:   datafmt.CSVConvertInput{Input: "", From: "csv", To: "json"},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name: "error path — invalid JSON array for json->csv",
			input: datafmt.CSVConvertInput{
				Input: `{"not":"an array"}`,
				From:  "json",
				To:    "csv",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name: "error path — unsupported conversion",
			input: datafmt.CSVConvertInput{
				Input: "a,b",
				From:  "csv",
				To:    "yaml",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := datafmt.CSVConvert(ctx, tc.input)
			tc.checkFn(t, result)
		})
	}
}

// ─── data_jsonpath ─────────────────────────────────────────────────────────────

func TestJSONPath(t *testing.T) {
	ctx := context.Background()

	doc := `{"store":{"book":[{"title":"Go Programming","price":29.99},{"title":"Rust Book","price":39.99}],"name":"bookstore"}}`

	tests := []struct {
		name    string
		input   datafmt.JSONPathInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "happy path — root field access",
			input: datafmt.JSONPathInput{
				JSON: doc,
				Path: "$.store.name",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["result"] != "bookstore" {
					t.Errorf("expected 'bookstore', got: %v", m["result"])
				}
			},
		},
		{
			name: "happy path — array index access",
			input: datafmt.JSONPathInput{
				JSON: doc,
				Path: "$.store.book[0].title",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["result"] != "Go Programming" {
					t.Errorf("expected 'Go Programming', got: %v", m["result"])
				}
			},
		},
		{
			name: "happy path — wildcard on array",
			input: datafmt.JSONPathInput{
				JSON: `{"items":[1,2,3]}`,
				Path: "$.items[*]",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				arr, ok := m["result"].([]any)
				if !ok {
					t.Fatalf("expected array result, got: %T — %v", m["result"], m["result"])
				}
				if len(arr) != 3 {
					t.Errorf("expected 3 items, got %d", len(arr))
				}
			},
		},
		{
			name: "happy path — wildcard on object",
			input: datafmt.JSONPathInput{
				JSON: `{"a":1,"b":2}`,
				Path: "$.*",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				arr, ok := m["result"].([]any)
				if !ok {
					t.Fatalf("expected array result, got: %T", m["result"])
				}
				if len(arr) != 2 {
					t.Errorf("expected 2 values, got %d", len(arr))
				}
			},
		},
		{
			name: "error path — missing $",
			input: datafmt.JSONPathInput{
				JSON: `{"a":1}`,
				Path: ".a",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name: "error path — invalid JSON",
			input: datafmt.JSONPathInput{
				JSON: `{bad}`,
				Path: "$.a",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name:    "error path — empty json",
			input:   datafmt.JSONPathInput{JSON: "", Path: "$.a"},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name:    "error path — empty path",
			input:   datafmt.JSONPathInput{JSON: `{"a":1}`, Path: ""},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := datafmt.JSONPath(ctx, tc.input)
			tc.checkFn(t, result)
		})
	}
}

// ─── data_schema_validate ─────────────────────────────────────────────────────

func TestSchemaValidate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   datafmt.SchemaValidateInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "happy path — valid document",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"name":"alice","age":30}`,
				Schema: `{"type":"object","required":["name","age"],"properties":{"name":{"type":"string"},"age":{"type":"number"}}}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != true {
					t.Errorf("expected valid=true, got: %s", result)
				}
			},
		},
		{
			name: "happy path — nested properties validated",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"user":{"name":"bob","score":95}}`,
				Schema: `{"type":"object","properties":{"user":{"type":"object","properties":{"name":{"type":"string"},"score":{"type":"number","minimum":0,"maximum":100}}}}}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != true {
					t.Errorf("expected valid=true, got: %s", result)
				}
			},
		},
		{
			name: "error path — missing required field",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"name":"alice"}`,
				Schema: `{"type":"object","required":["name","age"]}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != false {
					t.Errorf("expected valid=false, got: %s", result)
				}
				errs, ok := m["errors"].([]any)
				if !ok || len(errs) == 0 {
					t.Errorf("expected non-empty errors array, got: %s", result)
				}
			},
		},
		{
			name: "error path — wrong type",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"age":"thirty"}`,
				Schema: `{"type":"object","properties":{"age":{"type":"number"}}}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != false {
					t.Errorf("expected valid=false, got: %s", result)
				}
			},
		},
		{
			name: "error path — value below minimum",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"score":-5}`,
				Schema: `{"type":"object","properties":{"score":{"type":"number","minimum":0}}}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != false {
					t.Errorf("expected valid=false, got: %s", result)
				}
			},
		},
		{
			name: "error path — string too short",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"name":"ab"}`,
				Schema: `{"type":"object","properties":{"name":{"type":"string","minLength":3}}}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != false {
					t.Errorf("expected valid=false, got: %s", result)
				}
			},
		},
		{
			name: "error path — enum violation",
			input: datafmt.SchemaValidateInput{
				JSON:   `{"status":"unknown"}`,
				Schema: `{"type":"object","properties":{"status":{"type":"string","enum":["active","inactive"]}}}`,
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				if m["valid"] != false {
					t.Errorf("expected valid=false, got: %s", result)
				}
			},
		},
		{
			name:    "error path — invalid JSON document",
			input:   datafmt.SchemaValidateInput{JSON: `{bad}`, Schema: `{"type":"object"}`},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name:    "error path — invalid schema",
			input:   datafmt.SchemaValidateInput{JSON: `{}`, Schema: `{bad}`},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := datafmt.SchemaValidate(ctx, tc.input)
			tc.checkFn(t, result)
		})
	}
}

// ─── data_diff ────────────────────────────────────────────────────────────────

func TestDiff(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   datafmt.DiffInput
		checkFn func(t *testing.T, result string)
	}{
		{
			name: "happy path — added, removed, changed keys",
			input: datafmt.DiffInput{
				A:      `{"name":"alice","age":30,"city":"NY"}`,
				B:      `{"name":"alice","age":31,"country":"US"}`,
				Format: "json",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)

				added, _ := m["added"].([]any)
				if len(added) != 1 || added[0] != "country" {
					t.Errorf("expected added=[country], got: %v", added)
				}

				removed, _ := m["removed"].([]any)
				if len(removed) != 1 || removed[0] != "city" {
					t.Errorf("expected removed=[city], got: %v", removed)
				}

				changed, _ := m["changed"].([]any)
				if len(changed) != 1 {
					t.Errorf("expected 1 changed entry, got: %v", changed)
				}
				if entry, ok := changed[0].(map[string]any); ok {
					if entry["key"] != "age" {
						t.Errorf("expected changed key=age, got: %v", entry["key"])
					}
				}
			},
		},
		{
			name: "happy path — identical documents produce empty diff",
			input: datafmt.DiffInput{
				A:      `{"x":1,"y":2}`,
				B:      `{"x":1,"y":2}`,
				Format: "json",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				added, _ := m["added"].([]any)
				removed, _ := m["removed"].([]any)
				changed, _ := m["changed"].([]any)
				if len(added)+len(removed)+len(changed) != 0 {
					t.Errorf("expected empty diff for identical docs, got: %s", result)
				}
			},
		},
		{
			name: "happy path — YAML format diff",
			input: datafmt.DiffInput{
				A:      "x: 1\ny: 2\n",
				B:      "x: 1\ny: 3\n",
				Format: "yaml",
			},
			checkFn: func(t *testing.T, result string) {
				m := mustUnmarshal(t, result)
				changed, _ := m["changed"].([]any)
				if len(changed) != 1 {
					t.Errorf("expected 1 changed entry for YAML diff, got: %v", changed)
				}
			},
		},
		{
			name:    "error path — empty a",
			input:   datafmt.DiffInput{A: "", B: `{"x":1}`, Format: "json"},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name:    "error path — empty b",
			input:   datafmt.DiffInput{A: `{"x":1}`, B: "", Format: "json"},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name:    "error path — invalid JSON",
			input:   datafmt.DiffInput{A: `{bad}`, B: `{"x":1}`, Format: "json"},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
		{
			name: "error path — non-object JSON (array)",
			input: datafmt.DiffInput{
				A:      `[1,2,3]`,
				B:      `[4,5,6]`,
				Format: "json",
			},
			checkFn: func(t *testing.T, result string) { assertError(t, result) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := datafmt.Diff(ctx, tc.input)
			tc.checkFn(t, result)
		})
	}
}
