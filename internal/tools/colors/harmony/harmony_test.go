package harmony_test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"dev-forge-mcp/internal/tools/colors/harmony"
)

func TestCompute_AllHarmonyTypesReturnFiveHexColors(t *testing.T) {
	types := []string{
		"analogous",
		"monochromatic",
		"triad",
		"complementary",
		"split_complementary",
		"square",
		"compound",
		"shades",
	}

	hexRe := regexp.MustCompile(`^#[0-9A-F]{6}$`)

	for _, h := range types {
		h := h
		t.Run(h, func(t *testing.T) {
			out, err := harmony.Compute("#FF6B6B", h, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(out.Colors) != 5 {
				t.Fatalf("expected 5 colors, got %d", len(out.Colors))
			}
			for _, c := range out.Colors {
				if !hexRe.MatchString(c) {
					t.Fatalf("invalid hex color %q", c)
				}
			}
		})
	}
}

func TestCompute_UsesDefaultSpreadAndKeepsBaseColorNormalized(t *testing.T) {
	out, err := harmony.Compute("#ff6b6b", "analogous", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Spread != 30 {
		t.Fatalf("expected default spread 30, got %v", out.Spread)
	}
	if out.BaseColor != "#FF6B6B" {
		t.Fatalf("expected normalized base color #FF6B6B, got %s", out.BaseColor)
	}
	if out.Colors[2] != "#FF6B6B" {
		t.Fatalf("expected center analogous color to match base, got %s", out.Colors[2])
	}
}

func TestCompute_AcceptsAliasesAndCustomSpread(t *testing.T) {
	custom := 100.0
	out, err := harmony.Compute("#3366CC", "triadic", &custom)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Harmony != "triad" {
		t.Fatalf("expected normalized harmony triad, got %s", out.Harmony)
	}
	if out.Spread != custom {
		t.Fatalf("expected spread %v, got %v", custom, out.Spread)
	}
}

func TestCompute_RejectsInvalidInput(t *testing.T) {
	if _, err := harmony.Compute("", "analogous", nil); err == nil {
		t.Fatalf("expected error for empty base color")
	}
	if _, err := harmony.Compute("#GGGGGG", "analogous", nil); err == nil {
		t.Fatalf("expected error for invalid base color")
	}
	if _, err := harmony.Compute("#3366CC", "unknown", nil); err == nil {
		t.Fatalf("expected error for unsupported harmony")
	}
	neg := -15.0
	if _, err := harmony.Compute("#3366CC", "analogous", &neg); err == nil {
		t.Fatalf("expected error for negative spread")
	}
}

func TestGenerate_ReturnsErrorJSONOnInvalidInput(t *testing.T) {
	raw := harmony.Generate(context.Background(), harmony.GenerateInput{
		BaseColor: "#GGGGGG",
		Harmony:   "analogous",
	})

	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := out["error"]; !ok {
		t.Fatalf("expected error JSON, got: %s", raw)
	}
}
