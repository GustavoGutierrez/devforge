package micro_test

import (
	"context"
	"encoding/json"
	"testing"

	"dev-forge-mcp/internal/tools/frontend/micro"
)

func asMap(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	return m
}

func TestGenerateTextDiff(t *testing.T) {
	ctx := context.Background()
	raw := micro.GenerateTextDiff(ctx, micro.TextDiffInput{
		OriginalText: "status=active\n",
		ModifiedText: "status=inactive\n",
	})
	m := asMap(t, raw)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	diff := m["diff"].(string)
	if diff == "" {
		t.Fatalf("expected non-empty diff")
	}
}

func TestConvertCSSUnits(t *testing.T) {
	ctx := context.Background()
	raw := micro.ConvertCSSUnits(ctx, micro.CSSUnitsBatchInput{
		ValuesPX:   []float64{12, 16, 24},
		BaseSize:   16,
		TargetUnit: "rem",
	})
	m := asMap(t, raw)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	conv := m["conversions"].(map[string]any)
	if conv["16px"].(string) != "1rem" {
		t.Fatalf("expected 16px -> 1rem, got %v", conv["16px"])
	}
}

func TestCheckWCAGContrast(t *testing.T) {
	ctx := context.Background()
	raw := micro.CheckWCAGContrast(ctx, micro.WCAGContrastInput{
		ForegroundColor: "#000000",
		BackgroundColor: "#FFFFFF",
	})
	m := asMap(t, raw)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	ratio := m["contrast_ratio"].(float64)
	if ratio < 20 {
		t.Fatalf("expected high contrast ratio, got %v", ratio)
	}
	aa := m["wcag_aa"].(map[string]any)
	if aa["normal_text_pass"].(bool) != true {
		t.Fatalf("expected AA normal pass")
	}
}

func TestCalculateAspectRatio(t *testing.T) {
	ctx := context.Background()
	w := 1024.0
	raw := micro.CalculateAspectRatio(ctx, micro.AspectRatioInput{
		AspectRatio: "16:9",
		KnownWidth:  &w,
	})
	m := asMap(t, raw)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	h := m["height"].(float64)
	if h != 576 {
		t.Fatalf("expected height 576, got %v", h)
	}
}

func TestConvertStringCases(t *testing.T) {
	ctx := context.Background()
	raw := micro.ConvertStringCases(ctx, micro.StringCasesInput{
		Variables:  []string{"user_id", "created_at"},
		TargetCase: "camelCase",
	})
	m := asMap(t, raw)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error: %v", m["error"])
	}
	converted := m["converted"].(map[string]any)
	if converted["user_id"].(string) != "userId" {
		t.Fatalf("expected user_id -> userId, got %v", converted["user_id"])
	}
}

func TestErrors(t *testing.T) {
	ctx := context.Background()

	raw := micro.CalculateAspectRatio(ctx, micro.AspectRatioInput{})
	m := asMap(t, raw)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for empty aspect ratio input")
	}

	raw = micro.ConvertStringCases(ctx, micro.StringCasesInput{Variables: []string{"a"}, TargetCase: "unknown"})
	m = asMap(t, raw)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error for invalid target_case")
	}
}
