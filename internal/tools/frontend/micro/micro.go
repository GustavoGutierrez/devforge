// Package micro implements frontend/UI micro-utilities for LLM tool calling.
package micro

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"dev-forge-mcp/internal/tools/filetools"
	"dev-forge-mcp/internal/tools/frontend"
	"dev-forge-mcp/internal/tools/textenc"
)

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

// TextDiffInput defines the input for generate_text_diff.
type TextDiffInput struct {
	OriginalText string `json:"original_text"`
	ModifiedText string `json:"modified_text"`
}

// TextDiffOutput defines the output for generate_text_diff.
type TextDiffOutput struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// GenerateTextDiff compares two text blocks and returns a unified diff output.
func GenerateTextDiff(ctx context.Context, in TextDiffInput) string {
	raw := filetools.Diff(ctx, filetools.DiffInput{
		A:            in.OriginalText,
		B:            in.ModifiedText,
		Mode:         "text",
		ContextLines: 3,
	})

	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return errJSON("unexpected diff response")
	}
	if e, ok := m["error"].(string); ok && e != "" {
		return errJSON(e)
	}

	out := TextDiffOutput{}
	if v, ok := m["diff"].(string); ok {
		out.Diff = strings.ReplaceAll(strings.ReplaceAll(v, "--- a\n", "--- original.txt\n"), "+++ b\n", "+++ modified.txt\n")
	}
	if v, ok := m["additions"].(float64); ok {
		out.Additions = int(v)
	}
	if v, ok := m["deletions"].(float64); ok {
		out.Deletions = int(v)
	}

	return resultJSON(out)
}

// CSSUnitsBatchInput defines input for convert_css_units.
type CSSUnitsBatchInput struct {
	ValuesPX   []float64 `json:"values_px"`
	BaseSize   float64   `json:"base_size"`
	TargetUnit string    `json:"target_unit"` // rem | em
}

// CSSUnitsBatchOutput defines output for convert_css_units.
type CSSUnitsBatchOutput struct {
	Base        float64           `json:"base"`
	Unit        string            `json:"unit"`
	Conversions map[string]string `json:"conversions"`
}

// ConvertCSSUnits converts multiple pixel values to rem or em.
func ConvertCSSUnits(_ context.Context, in CSSUnitsBatchInput) string {
	if len(in.ValuesPX) == 0 {
		return errJSON("values_px is required")
	}
	base := in.BaseSize
	if base <= 0 {
		base = 16
	}
	unit := strings.ToLower(strings.TrimSpace(in.TargetUnit))
	if unit == "" {
		unit = "rem"
	}
	if unit != "rem" && unit != "em" {
		return errJSON("target_unit must be rem or em")
	}

	conv := make(map[string]string, len(in.ValuesPX))
	for _, v := range in.ValuesPX {
		res := v / base
		conv[fmt.Sprintf("%spx", trimFloat(v))] = fmt.Sprintf("%s%s", trimFloat(res), unit)
	}

	return resultJSON(CSSUnitsBatchOutput{
		Base:        base,
		Unit:        unit,
		Conversions: conv,
	})
}

type wcagLevelResult struct {
	NormalTextPass bool `json:"normal_text_pass"`
	LargeTextPass  bool `json:"large_text_pass"`
}

// WCAGContrastInput defines input for check_wcag_contrast.
type WCAGContrastInput struct {
	ForegroundColor string `json:"foreground_color"`
	BackgroundColor string `json:"background_color"`
}

// WCAGContrastOutput defines output for check_wcag_contrast.
type WCAGContrastOutput struct {
	ContrastRatio float64         `json:"contrast_ratio"`
	WCAGAA        wcagLevelResult `json:"wcag_aa"`
	WCAGAAA       wcagLevelResult `json:"wcag_aaa"`
}

// CheckWCAGContrast computes WCAG contrast and pass/fail for AA/AAA, normal and large text.
func CheckWCAGContrast(ctx context.Context, in WCAGContrastInput) string {
	if strings.TrimSpace(in.ForegroundColor) == "" || strings.TrimSpace(in.BackgroundColor) == "" {
		return errJSON("foreground_color and background_color are required")
	}

	raw := frontend.Color(ctx, frontend.ColorInput{
		Color:   in.ForegroundColor,
		To:      "hex",
		Against: in.BackgroundColor,
	})

	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return errJSON("unexpected color response")
	}
	if e, ok := m["error"].(string); ok && e != "" {
		return errJSON(e)
	}
	ratio, ok := m["contrast_ratio"].(float64)
	if !ok {
		return errJSON("could not compute contrast ratio")
	}

	out := WCAGContrastOutput{
		ContrastRatio: ratio,
		WCAGAA: wcagLevelResult{
			NormalTextPass: ratio >= 4.5,
			LargeTextPass:  ratio >= 3.0,
		},
		WCAGAAA: wcagLevelResult{
			NormalTextPass: ratio >= 7.0,
			LargeTextPass:  ratio >= 4.5,
		},
	}
	return resultJSON(out)
}

// AspectRatioInput defines input for calculate_aspect_ratio.
type AspectRatioInput struct {
	AspectRatio string   `json:"aspect_ratio,omitempty"`
	KnownWidth  *float64 `json:"known_width,omitempty"`
	KnownHeight *float64 `json:"known_height,omitempty"`
}

// AspectRatioOutput defines output for calculate_aspect_ratio.
type AspectRatioOutput struct {
	AspectRatio  string  `json:"aspect_ratio"`
	RatioDecimal float64 `json:"ratio_decimal"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
}

// CalculateAspectRatio computes a missing dimension or deduces ratio from known dimensions.
func CalculateAspectRatio(_ context.Context, in AspectRatioInput) string {
	var ratio float64
	var ratioLabel string

	if strings.TrimSpace(in.AspectRatio) != "" {
		rw, rh, err := parseRatio(in.AspectRatio)
		if err != nil {
			return errJSON(err.Error())
		}
		ratio = rw / rh
		ratioLabel = fmt.Sprintf("%d:%d", int(math.Round(rw)), int(math.Round(rh)))
	}

	hasW := in.KnownWidth != nil && *in.KnownWidth > 0
	hasH := in.KnownHeight != nil && *in.KnownHeight > 0

	if !hasW && !hasH {
		return errJSON("known_width or known_height is required")
	}

	var width, height float64
	if hasW {
		width = *in.KnownWidth
	}
	if hasH {
		height = *in.KnownHeight
	}

	if ratio == 0 {
		if !hasW || !hasH {
			return errJSON("aspect_ratio is required when one dimension is missing")
		}
		ratio = width / height
		ratioLabel = simplifyRatio(width, height)
	} else {
		if hasW && !hasH {
			height = math.Round((width/ratio)*10000) / 10000
		} else if hasH && !hasW {
			width = math.Round((height*ratio)*10000) / 10000
		}
	}

	out := AspectRatioOutput{
		AspectRatio:  ratioLabel,
		RatioDecimal: math.Round(ratio*10000) / 10000,
		Width:        width,
		Height:       height,
	}
	return resultJSON(out)
}

// StringCasesInput defines input for convert_string_cases.
type StringCasesInput struct {
	Variables  []string `json:"variables"`
	TargetCase string   `json:"target_case"`
}

// StringCasesOutput defines output for convert_string_cases.
type StringCasesOutput struct {
	Converted map[string]string `json:"converted"`
}

// ConvertStringCases converts a list of variable names to the desired case format.
func ConvertStringCases(ctx context.Context, in StringCasesInput) string {
	if len(in.Variables) == 0 {
		return errJSON("variables is required")
	}
	target, err := normalizeTargetCase(in.TargetCase)
	if err != nil {
		return errJSON(err.Error())
	}

	converted := make(map[string]string, len(in.Variables))
	for _, v := range in.Variables {
		raw := textenc.Case(ctx, textenc.CaseInput{Text: v, TargetCase: target})
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return errJSON("unexpected case conversion response")
		}
		if e, ok := m["error"].(string); ok && e != "" {
			return errJSON(e)
		}
		res, _ := m["result"].(string)
		converted[v] = res
	}

	return resultJSON(StringCasesOutput{Converted: converted})
}

func normalizeTargetCase(v string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")

	switch s {
	case "camel", "camelcase":
		return "camel", nil
	case "snake", "snakecase":
		return "snake", nil
	case "kebab", "kebabcase":
		return "kebab", nil
	case "pascal", "pascalcase":
		return "pascal", nil
	case "screamingsnake", "screamingsnakecase":
		return "screaming_snake", nil
	default:
		return "", fmt.Errorf("target_case must be camelCase, snake_case, kebab-case, or PascalCase")
	}
}

func parseRatio(v string) (float64, float64, error) {
	parts := strings.Split(strings.TrimSpace(v), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("aspect_ratio must be in W:H format")
	}
	w, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("invalid aspect_ratio width")
	}
	h, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("invalid aspect_ratio height")
	}
	return w, h, nil
}

func simplifyRatio(w, h float64) string {
	wi := int(math.Round(w))
	hi := int(math.Round(h))
	if wi <= 0 || hi <= 0 {
		return "1:1"
	}
	g := gcd(wi, hi)
	return fmt.Sprintf("%d:%d", wi/g, hi/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	if a == 0 {
		return 1
	}
	return a
}

func trimFloat(v float64) string {
	r := math.Round(v*10000) / 10000
	s := strconv.FormatFloat(r, 'f', 4, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "0"
	}
	return s
}
