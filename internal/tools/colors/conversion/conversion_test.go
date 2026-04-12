package conversion_test

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/colors/conversion"
)

func TestConvert_HexToRGB(t *testing.T) {
	out, err := conversion.Compute(conversion.ConvertInput{
		Color: "#FF0000",
		From:  "hex",
		To:    "rgb",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Result != "rgb(255, 0, 0)" {
		t.Fatalf("unexpected result: %s", out.Result)
	}
}

func TestConvert_RGBToHex(t *testing.T) {
	out, err := conversion.Compute(conversion.ConvertInput{
		Color: "rgb(59, 130, 246)",
		From:  "rgb",
		To:    "hex",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Result != "#3B82F6" {
		t.Fatalf("unexpected result: %s", out.Result)
	}
}

func TestConvert_HexToLABAndOKLCH(t *testing.T) {
	labOut, err := conversion.Compute(conversion.ConvertInput{
		Color: "#FF0000",
		From:  "hex",
		To:    "lab",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertApprox(t, labOut.Components["l"], 53.24, 0.2)
	assertApprox(t, labOut.Components["a"], 80.09, 0.3)
	assertApprox(t, labOut.Components["b"], 67.20, 0.3)

	okOut, err := conversion.Compute(conversion.ConvertInput{
		Color: "#FF0000",
		From:  "hex",
		To:    "oklch",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertApprox(t, okOut.Components["l"], 0.628, 0.01)
	assertApprox(t, okOut.Components["h"], 29.2, 1.0)
}

func TestConvert_HSLInput_WithPercentSyntax(t *testing.T) {
	out, err := conversion.Compute(conversion.ConvertInput{
		Color: "hsl(0, 100%, 50%)",
		From:  "hsl",
		To:    "hex",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Result != "#FF0000" {
		t.Fatalf("expected #FF0000, got %s", out.Result)
	}
}

func TestConvert_AliasLinearSRGB(t *testing.T) {
	out, err := conversion.Compute(conversion.ConvertInput{
		Color: "#808080",
		From:  "hex",
		To:    "linear-srgb",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out.Result, "linear_rgb(") {
		t.Fatalf("unexpected result format: %s", out.Result)
	}
}

func TestConvert_RoundTripHexThroughLAB(t *testing.T) {
	labOut, err := conversion.Compute(conversion.ConvertInput{
		Color: "#3B82F6",
		From:  "hex",
		To:    "lab",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backOut, err := conversion.Compute(conversion.ConvertInput{
		Color: labOut.Result,
		From:  "lab",
		To:    "hex",
	})
	if err != nil {
		t.Fatalf("unexpected error on back conversion: %v", err)
	}

	if backOut.Result != "#3B82F6" {
		t.Fatalf("expected #3B82F6 after round-trip, got %s", backOut.Result)
	}
}

func TestConvert_RejectsInvalidInput(t *testing.T) {
	_, err := conversion.Compute(conversion.ConvertInput{Color: "", From: "hex", To: "rgb"})
	if err == nil {
		t.Fatalf("expected error for empty color")
	}

	_, err = conversion.Compute(conversion.ConvertInput{Color: "#GGGGGG", From: "hex", To: "rgb"})
	if err == nil {
		t.Fatalf("expected error for invalid hex")
	}

	_, err = conversion.Compute(conversion.ConvertInput{Color: "#FFFFFF", From: "hex", To: "unknown"})
	if err == nil {
		t.Fatalf("expected error for unsupported destination space")
	}
}

func TestConvert_JSONErrorWrapper(t *testing.T) {
	raw := conversion.Convert(context.Background(), conversion.ConvertInput{
		Color: "#GGGGGG",
		From:  "hex",
		To:    "rgb",
	})

	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := out["error"]; !ok {
		t.Fatalf("expected error JSON, got: %s", raw)
	}
}

func assertApprox(t *testing.T, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("expected %.6f ± %.6f, got %.6f", want, tol, got)
	}
}
