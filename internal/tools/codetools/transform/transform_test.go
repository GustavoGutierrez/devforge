package transform_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/codetools/transform"
)

func decodeJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	return m
}

func TestJSONToTypes(t *testing.T) {
	ctx := context.Background()
	input := `{"id":1,"name":"Ada","active":true,"address":{"city":"Quito"}}`

	// TypeScript
	ts := transform.JSONToTypes(ctx, transform.JSONToTypesInput{JSON: input, Language: "typescript", RootName: "user"})
	tsMap := decodeJSON(t, ts)
	if _, ok := tsMap["error"]; ok {
		t.Fatalf("unexpected error: %s", ts)
	}
	code := tsMap["code"].(string)
	if !strings.Contains(code, "export interface User") {
		t.Fatalf("expected root TS interface, got: %s", code)
	}

	// Go
	goOut := transform.JSONToTypes(ctx, transform.JSONToTypesInput{JSON: input, Language: "go", RootName: "user"})
	goMap := decodeJSON(t, goOut)
	if _, ok := goMap["error"]; ok {
		t.Fatalf("unexpected error: %s", goOut)
	}
	goCode := goMap["code"].(string)
	if !strings.Contains(goCode, "type User struct") {
		t.Fatalf("expected root Go struct, got: %s", goCode)
	}
	if !strings.Contains(goCode, "`json:\"name\"`") {
		t.Fatalf("expected json tags in Go struct, got: %s", goCode)
	}

	// Rust
	rustOut := transform.JSONToTypes(ctx, transform.JSONToTypesInput{JSON: input, Language: "rust", RootName: "user"})
	rustMap := decodeJSON(t, rustOut)
	if _, ok := rustMap["error"]; ok {
		t.Fatalf("unexpected error: %s", rustOut)
	}
	rustCode := rustMap["code"].(string)
	if !strings.Contains(rustCode, "pub struct User") {
		t.Fatalf("expected root Rust struct, got: %s", rustCode)
	}
	if !strings.Contains(rustCode, "use serde::{Deserialize, Serialize};") {
		t.Fatalf("expected serde imports, got: %s", rustCode)
	}
}

func TestJSONToTypesErrors(t *testing.T) {
	ctx := context.Background()

	raw := transform.JSONToTypes(ctx, transform.JSONToTypesInput{JSON: "", Language: "go"})
	m := decodeJSON(t, raw)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for empty input")
	}

	raw = transform.JSONToTypes(ctx, transform.JSONToTypesInput{JSON: "{bad", Language: "go"})
	m = decodeJSON(t, raw)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for invalid json")
	}
}

func TestASTExplore(t *testing.T) {
	ctx := context.Background()
	code := `import x from "mod"
export class UserService {
  constructor() {}
}
function ping() { return true }
const handler = () => { return 1 }
`

	raw := transform.ASTExplore(ctx, transform.ASTExploreInput{Code: code, Language: "typescript"})
	m := decodeJSON(t, raw)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %s", raw)
	}

	summary := m["summary"].(map[string]any)
	if summary["ClassDeclaration"].(float64) < 1 {
		t.Fatalf("expected class declaration in summary: %v", summary)
	}
	if summary["FunctionDeclaration"].(float64) < 1 {
		t.Fatalf("expected function declaration in summary: %v", summary)
	}
	if summary["ArrowFunctionVariable"].(float64) < 1 {
		t.Fatalf("expected arrow function variable in summary: %v", summary)
	}
}

func TestASTExploreErrors(t *testing.T) {
	ctx := context.Background()
	raw := transform.ASTExplore(ctx, transform.ASTExploreInput{Code: "", Language: "javascript"})
	m := decodeJSON(t, raw)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for empty code")
	}
}
