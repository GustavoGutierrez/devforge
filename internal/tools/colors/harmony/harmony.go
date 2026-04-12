// Package harmony provides color harmony palette generation utilities.
//
// The package is intentionally isolated under internal/tools/colors so new
// color-related modules can be added without coupling to frontend helpers.
package harmony

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// GenerateInput is the MCP input schema for color harmony generation.
type GenerateInput struct {
	BaseColor string   `json:"base_color"`
	Harmony   string   `json:"harmony"`
	Spread    *float64 `json:"spread,omitempty"`
}

// GenerateOutput is the MCP output schema for color harmony generation.
type GenerateOutput struct {
	BaseColor string   `json:"base_color"`
	Harmony   string   `json:"harmony"`
	Spread    float64  `json:"spread"`
	Colors    []string `json:"colors"`
}

type rgb struct {
	R uint8
	G uint8
	B uint8
}

type hsv struct {
	H float64 // 0..360
	S float64 // 0..1
	V float64 // 0..1
}

// Generate computes a 5-color harmony palette and returns a JSON payload.
func Generate(_ context.Context, in GenerateInput) string {
	out, err := Compute(in.BaseColor, in.Harmony, in.Spread)
	if err != nil {
		return errJSON(err.Error())
	}
	return resultJSON(out)
}

// Compute computes a 5-color palette for the requested harmony type.
func Compute(baseColor, harmonyType string, spread *float64) (GenerateOutput, error) {
	if strings.TrimSpace(baseColor) == "" {
		return GenerateOutput{}, fmt.Errorf("base_color is required")
	}
	if strings.TrimSpace(harmonyType) == "" {
		return GenerateOutput{}, fmt.Errorf("harmony is required")
	}

	normalizedHarmony, err := normalizeHarmony(harmonyType)
	if err != nil {
		return GenerateOutput{}, err
	}

	resolvedSpread := defaultSpread(normalizedHarmony)
	if spread != nil {
		if *spread < 0 {
			return GenerateOutput{}, fmt.Errorf("spread must be >= 0")
		}
		resolvedSpread = *spread
	}

	baseRGB, err := hexToRGB(baseColor)
	if err != nil {
		return GenerateOutput{}, fmt.Errorf("invalid base_color: %w", err)
	}
	baseHex := rgbToHex(baseRGB)
	baseHSV := rgbToHSV(baseRGB)

	paletteHSV := buildPalette(baseHSV, normalizedHarmony, resolvedSpread)
	colors := make([]string, 0, len(paletteHSV))
	for _, c := range paletteHSV {
		colors = append(colors, rgbToHex(hsvToRGB(c)))
	}

	return GenerateOutput{
		BaseColor: baseHex,
		Harmony:   normalizedHarmony,
		Spread:    resolvedSpread,
		Colors:    colors,
	}, nil
}

func buildPalette(base hsv, harmonyType string, spread float64) []hsv {
	shift := func(h, amount float64) float64 { return normalizeHue(h + amount) }
	clamp01 := func(v float64) float64 { return clamp(v, 0, 1) }

	switch harmonyType {
	case "analogous":
		return []hsv{
			{H: shift(base.H, -spread*2), S: base.S, V: base.V},
			{H: shift(base.H, -spread), S: base.S, V: base.V},
			base,
			{H: shift(base.H, spread), S: base.S, V: base.V},
			{H: shift(base.H, spread*2), S: base.S, V: base.V},
		}
	case "monochromatic":
		return []hsv{
			{H: base.H, S: clamp01(base.S * 0.5), V: clamp01(base.V * 1.0)},
			{H: base.H, S: clamp01(base.S * 0.7), V: clamp01(base.V * 0.8)},
			base,
			{H: base.H, S: clamp01(base.S * 1.0), V: clamp01(base.V * 0.6)},
			{H: base.H, S: clamp01(base.S * 1.0), V: clamp01(base.V * 0.4)},
		}
	case "triad":
		return []hsv{
			base,
			{H: shift(base.H, spread), S: base.S, V: base.V},
			{H: shift(base.H, spread*2), S: base.S, V: base.V},
			{H: base.H, S: base.S, V: clamp01(base.V * 0.7)},
			{H: shift(base.H, spread), S: base.S, V: clamp01(base.V * 0.7)},
		}
	case "complementary":
		compH := shift(base.H, 180)
		return []hsv{
			{H: base.H, S: clamp01(base.S * 0.7), V: base.V},
			base,
			{H: compH, S: clamp01(base.S * 0.5), V: clamp01(base.V * 0.8)},
			{H: compH, S: base.S, V: base.V},
			{H: compH, S: base.S, V: clamp01(base.V * 0.6)},
		}
	case "split_complementary":
		return []hsv{
			base,
			{H: base.H, S: clamp01(base.S * 0.5), V: base.V},
			{H: shift(base.H, spread), S: base.S, V: base.V},
			{H: shift(base.H, 360-spread), S: base.S, V: base.V},
			{H: shift(base.H, spread), S: clamp01(base.S * 0.6), V: base.V},
		}
	case "square":
		return []hsv{
			base,
			{H: shift(base.H, spread), S: base.S, V: base.V},
			{H: shift(base.H, spread*2), S: base.S, V: base.V},
			{H: shift(base.H, spread*3), S: base.S, V: base.V},
			{H: base.H, S: clamp01(base.S * 0.5), V: clamp01(base.V * 0.8)},
		}
	case "compound":
		return []hsv{
			base,
			{H: shift(base.H, spread), S: base.S, V: base.V},
			{H: shift(base.H, 180-spread), S: base.S, V: base.V},
			{H: shift(base.H, 180), S: base.S, V: base.V},
			{H: shift(base.H, 180+spread), S: base.S, V: base.V},
		}
	case "shades":
		return []hsv{
			{H: base.H, S: base.S, V: clamp01(base.V * 1.0)},
			{H: base.H, S: base.S, V: clamp01(base.V * 0.8)},
			{H: base.H, S: base.S, V: clamp01(base.V * 0.6)},
			{H: base.H, S: base.S, V: clamp01(base.V * 0.4)},
			{H: base.H, S: base.S, V: clamp01(base.V * 0.2)},
		}
	default:
		return []hsv{base}
	}
}

func normalizeHarmony(h string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(h))
	n = strings.ReplaceAll(n, "-", "_")
	n = strings.ReplaceAll(n, " ", "_")

	aliases := map[string]string{
		"triadic":            "triad",
		"splitcomplementary": "split_complementary",
		"split_complement":   "split_complementary",
		"monochrome":         "monochromatic",
		"complement":         "complementary",
	}
	if canonical, ok := aliases[n]; ok {
		n = canonical
	}

	supported := map[string]bool{
		"analogous":           true,
		"monochromatic":       true,
		"triad":               true,
		"complementary":       true,
		"split_complementary": true,
		"square":              true,
		"compound":            true,
		"shades":              true,
	}
	if !supported[n] {
		return "", fmt.Errorf("unsupported harmony %q", h)
	}
	return n, nil
}

func defaultSpread(harmony string) float64 {
	switch harmony {
	case "analogous":
		return 30
	case "triad":
		return 120
	case "split_complementary":
		return 150
	case "square":
		return 90
	case "compound":
		return 30
	default:
		return 0
	}
}

func normalizeHue(h float64) float64 {
	v := math.Mod(h, 360)
	if v < 0 {
		v += 360
	}
	return v
}

func hexToRGB(hex string) (rgb, error) {
	hex = strings.TrimSpace(strings.TrimPrefix(hex, "#"))
	if len(hex) == 3 {
		hex = strings.Repeat(string(hex[0]), 2) +
			strings.Repeat(string(hex[1]), 2) +
			strings.Repeat(string(hex[2]), 2)
	}
	if len(hex) != 6 {
		return rgb{}, fmt.Errorf("hex color must be #RGB or #RRGGBB")
	}

	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("invalid hex value")
	}
	return rgb{R: uint8(v >> 16), G: uint8((v >> 8) & 0xFF), B: uint8(v & 0xFF)}, nil
}

func rgbToHex(c rgb) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func rgbToHSV(c rgb) hsv {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	maxC := math.Max(r, math.Max(g, b))
	minC := math.Min(r, math.Min(g, b))
	delta := maxC - minC

	h := 0.0
	s := 0.0
	v := maxC

	if delta > 0 {
		s = delta / maxC
		switch maxC {
		case r:
			h = math.Mod((g-b)/delta, 6)
		case g:
			h = ((b-r)/delta + 2)
		case b:
			h = ((r-g)/delta + 4)
		}
		h *= 60
		h = normalizeHue(h)
	}

	return hsv{H: h, S: s, V: v}
}

func hsvToRGB(c hsv) rgb {
	h := normalizeHue(c.H)
	s := clamp(c.S, 0, 1)
	v := clamp(c.V, 0, 1)

	chroma := v * s
	x := chroma * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - chroma

	var r1, g1, b1 float64
	switch {
	case h >= 0 && h < 60:
		r1, g1, b1 = chroma, x, 0
	case h >= 60 && h < 120:
		r1, g1, b1 = x, chroma, 0
	case h >= 120 && h < 180:
		r1, g1, b1 = 0, chroma, x
	case h >= 180 && h < 240:
		r1, g1, b1 = 0, x, chroma
	case h >= 240 && h < 300:
		r1, g1, b1 = x, 0, chroma
	default:
		r1, g1, b1 = chroma, 0, x
	}

	return rgb{
		R: uint8(math.Round((r1 + m) * 255)),
		G: uint8(math.Round((g1 + m) * 255)),
		B: uint8(math.Round((b1 + m) * 255)),
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func errJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errJSON("marshal failed: " + err.Error())
	}
	return string(b)
}
