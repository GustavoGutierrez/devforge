package gradient_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/colors/gradient"
)

func TestCompute_LinearTwoColorsAutoPositions(t *testing.T) {
	out, err := gradient.Compute(gradient.GenerateInput{
		GradientType: "linear",
		Stops: []gradient.ColorStopInput{
			{Color: "#22c1c3"},
			{Color: "#fdbb2d"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Gradient != "linear-gradient(0deg, #22c1c3, #fdbb2d)" {
		t.Fatalf("unexpected gradient: %s", out.Gradient)
	}
	if !strings.Contains(out.CSS, "background: #22c1c3;") {
		t.Fatalf("expected fallback in css, got: %s", out.CSS)
	}
}

func TestCompute_RadialWithShapeAndPositions(t *testing.T) {
	p0 := 0
	p100 := 100
	out, err := gradient.Compute(gradient.GenerateInput{
		GradientType: "radial",
		Shape:        "circle",
		Stops: []gradient.ColorStopInput{
			{Color: "rgba(34, 193, 195, 1)", Position: &p0},
			{Color: "rgba(253, 187, 45, 1)", Position: &p100},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "radial-gradient(circle, rgba(34, 193, 195, 1) 0%, rgba(253, 187, 45, 1) 100%)"
	if out.Gradient != want {
		t.Fatalf("unexpected gradient:\n got: %s\nwant: %s", out.Gradient, want)
	}
}

func TestCompute_ClampStopPosition(t *testing.T) {
	neg := -20
	over := 120
	out, err := gradient.Compute(gradient.GenerateInput{
		GradientType: "linear",
		Stops: []gradient.ColorStopInput{
			{Color: "#000", Position: &neg},
			{Color: "#fff", Position: &over},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *out.Stops[0].Position != 0 || *out.Stops[1].Position != 100 {
		t.Fatalf("expected clamped positions 0 and 100, got %d and %d", *out.Stops[0].Position, *out.Stops[1].Position)
	}
}

func TestCompute_ValidationErrors(t *testing.T) {
	_, err := gradient.Compute(gradient.GenerateInput{GradientType: "", Stops: nil})
	if err == nil {
		t.Fatalf("expected error for missing gradient_type")
	}

	_, err = gradient.Compute(gradient.GenerateInput{GradientType: "linear", Stops: []gradient.ColorStopInput{{Color: "#fff"}}})
	if err == nil {
		t.Fatalf("expected error for less than 2 stops")
	}

	_, err = gradient.Compute(gradient.GenerateInput{GradientType: "radial", Shape: "triangle", Stops: []gradient.ColorStopInput{{Color: "#000"}, {Color: "#fff"}}})
	if err == nil {
		t.Fatalf("expected error for invalid radial shape")
	}
}

func TestGenerate_ErrorJSON(t *testing.T) {
	raw := gradient.Generate(context.Background(), gradient.GenerateInput{
		GradientType: "linear",
		Stops:        []gradient.ColorStopInput{{Color: "#fff"}},
	})

	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := out["error"]; !ok {
		t.Fatalf("expected error JSON, got: %s", raw)
	}
}
