// Package frontend implements MCP tools for frontend development utilities.
// All functions are stateless and safe for concurrent use.
package frontend

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// errResult returns a JSON-encoded {"error": "..."} string.
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

// ── frontend_color ────────────────────────────────────────────────────────────

// ColorInput holds the parameters for the frontend_color tool.
type ColorInput struct {
	Color   string  // hex #RRGGBB, rgb(r,g,b), or hsl(h,s%,l%)
	To      string  // hex | rgb | hsl | hsla | rgba, default "hex"
	Alpha   float64 // default 1.0
	Against string  // optional second color for contrast ratio
}

// ColorResult is the JSON response for the frontend_color tool.
type ColorResult struct {
	Result        string   `json:"result"`
	ContrastRatio *float64 `json:"contrast_ratio,omitempty"`
	WCAGAA        *bool    `json:"wcag_aa,omitempty"`
	WCAGAAA       *bool    `json:"wcag_aaa,omitempty"`
}

// rgb holds a color as linear 0-255 red, green, blue values.
type rgb struct{ r, g, b float64 }

// Color converts a color string between formats and optionally computes WCAG contrast.
func Color(_ context.Context, in ColorInput) string {
	if in.Color == "" {
		return errResult("color is required")
	}
	to := in.To
	if to == "" {
		to = "hex"
	}
	alpha := in.Alpha
	if alpha == 0 {
		alpha = 1.0
	}

	c, err := parseColor(in.Color)
	if err != nil {
		return errResult("invalid color: " + err.Error())
	}

	result := formatColor(c, to, alpha)

	res := ColorResult{Result: result}

	if in.Against != "" {
		c2, err := parseColor(in.Against)
		if err != nil {
			return errResult("invalid against color: " + err.Error())
		}
		ratio := contrastRatio(c, c2)
		rounded := math.Round(ratio*100) / 100
		res.ContrastRatio = &rounded

		aa := ratio >= 4.5
		aaa := ratio >= 7.0
		res.WCAGAA = &aa
		res.WCAGAAA = &aaa
	}

	return resultJSON(res)
}

// parseColor parses hex, rgb(), or hsl() strings into an rgb struct.
func parseColor(s string) (rgb, error) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "#"):
		return parseHex(s)
	case strings.HasPrefix(s, "rgb("):
		return parseRGBFunc(s)
	case strings.HasPrefix(s, "hsl("):
		return parseHSLFunc(s)
	default:
		return rgb{}, fmt.Errorf("unsupported format: %q (use #RRGGBB, rgb(), or hsl())", s)
	}
}

// parseHex parses #RGB or #RRGGBB.
func parseHex(s string) (rgb, error) {
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
		// ok
	default:
		return rgb{}, fmt.Errorf("hex color must be #RGB or #RRGGBB")
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("invalid hex value: %v", err)
	}
	return rgb{
		r: float64((n >> 16) & 0xFF),
		g: float64((n >> 8) & 0xFF),
		b: float64(n & 0xFF),
	}, nil
}

// parseRGBFunc parses rgb(r, g, b) — values 0-255.
func parseRGBFunc(s string) (rgb, error) {
	inner := extractInner(s, "rgb(", ")")
	parts := splitCSV(inner)
	if len(parts) < 3 {
		return rgb{}, fmt.Errorf("rgb() requires 3 components")
	}
	vals, err := parseFloats(parts[:3])
	if err != nil {
		return rgb{}, err
	}
	return rgb{r: clamp(vals[0], 0, 255), g: clamp(vals[1], 0, 255), b: clamp(vals[2], 0, 255)}, nil
}

// parseHSLFunc parses hsl(h, s%, l%) — h 0-360, s/l 0-100.
func parseHSLFunc(s string) (rgb, error) {
	inner := extractInner(s, "hsl(", ")")
	parts := splitCSV(inner)
	if len(parts) < 3 {
		return rgb{}, fmt.Errorf("hsl() requires 3 components")
	}
	h, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return rgb{}, fmt.Errorf("invalid hue: %v", err)
	}
	sl := strings.TrimSuffix(strings.TrimSpace(parts[1]), "%")
	ll := strings.TrimSuffix(strings.TrimSpace(parts[2]), "%")
	sv, err1 := strconv.ParseFloat(sl, 64)
	lv, err2 := strconv.ParseFloat(ll, 64)
	if err1 != nil || err2 != nil {
		return rgb{}, fmt.Errorf("invalid saturation or lightness")
	}
	return hslToRGB(h, sv/100, lv/100), nil
}

// hslToRGB converts HSL (h: 0-360, s: 0-1, l: 0-1) to RGB (0-255).
func hslToRGB(h, s, l float64) rgb {
	if s == 0 {
		v := l * 255
		return rgb{r: v, g: v, b: v}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	h /= 360
	return rgb{
		r: math.Round(hue2rgb(p, q, h+1.0/3) * 255),
		g: math.Round(hue2rgb(p, q, h) * 255),
		b: math.Round(hue2rgb(p, q, h-1.0/3) * 255),
	}
}

func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 1.0/2:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	default:
		return p
	}
}

// rgbToHSL converts RGB (0-255) to HSL (h: 0-360, s: 0-1, l: 0-1).
func rgbToHSL(c rgb) (h, s, l float64) {
	r, g, b := c.r/255, c.g/255, c.b/255
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	l = (max + min) / 2
	if max == min {
		return 0, 0, l
	}
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	case b:
		h = (r-g)/d + 4
	}
	h *= 60
	return h, s, l
}

// formatColor converts rgb to the target format with given alpha.
func formatColor(c rgb, to string, alpha float64) string {
	r, g, b := int(math.Round(c.r)), int(math.Round(c.g)), int(math.Round(c.b))
	switch to {
	case "hex":
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	case "rgb":
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
	case "rgba":
		return fmt.Sprintf("rgba(%d, %d, %d, %s)", r, g, b, formatAlpha(alpha))
	case "hsl":
		h, s, l := rgbToHSL(c)
		return fmt.Sprintf("hsl(%s, %s%%, %s%%)",
			formatDegree(h), formatPct(s*100), formatPct(l*100))
	case "hsla":
		h, s, l := rgbToHSL(c)
		return fmt.Sprintf("hsla(%s, %s%%, %s%%, %s)",
			formatDegree(h), formatPct(s*100), formatPct(l*100), formatAlpha(alpha))
	default:
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
}

func formatAlpha(a float64) string {
	if a == float64(int(a)) {
		return strconv.Itoa(int(a))
	}
	return strconv.FormatFloat(a, 'f', 2, 64)
}

func formatDegree(v float64) string {
	rounded := math.Round(v*10) / 10
	if rounded == float64(int(rounded)) {
		return strconv.Itoa(int(rounded))
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

func formatPct(v float64) string {
	rounded := math.Round(v*10) / 10
	if rounded == float64(int(rounded)) {
		return strconv.Itoa(int(rounded))
	}
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}

// linearize converts a sRGB 0-255 component to linear light (for WCAG).
func linearize(c float64) float64 {
	s := c / 255
	if s <= 0.04045 {
		return s / 12.92
	}
	return math.Pow((s+0.055)/1.055, 2.4)
}

// relativeLuminance returns the WCAG 2.1 relative luminance of a color.
func relativeLuminance(c rgb) float64 {
	r := linearize(c.r)
	g := linearize(c.g)
	b := linearize(c.b)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// contrastRatio computes WCAG 2.1 contrast ratio between two colors.
func contrastRatio(c1, c2 rgb) float64 {
	l1 := relativeLuminance(c1)
	l2 := relativeLuminance(c2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// extractInner removes the prefix and suffix from s.
func extractInner(s, prefix, suffix string) string {
	s = strings.TrimPrefix(s, prefix)
	s = strings.TrimSuffix(s, suffix)
	return s
}

// splitCSV splits a comma-separated string.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// parseFloats parses a slice of strings as float64.
func parseFloats(parts []string) ([]float64, error) {
	out := make([]float64, len(parts))
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %v", p, err)
		}
		out[i] = v
	}
	return out, nil
}

// clamp restricts a value to [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ── frontend_css_unit ─────────────────────────────────────────────────────────

// CSSUnitInput holds the parameters for the frontend_css_unit tool.
type CSSUnitInput struct {
	Value          float64
	From           string  // px | rem | em | percent | vw | vh
	To             string  // px | rem | em | percent | vw | vh
	BaseFontSize   float64 // default 16
	ViewportWidth  float64 // default 1440
	ViewportHeight float64 // default 900
	ParentSize     float64 // default 16 — relative context for em/percent
}

// CSSUnit converts a CSS value between units.
func CSSUnit(_ context.Context, in CSSUnitInput) string {
	if in.From == "" || in.To == "" {
		return errResult("from and to are required")
	}
	base := in.BaseFontSize
	if base <= 0 {
		base = 16
	}
	vw := in.ViewportWidth
	if vw <= 0 {
		vw = 1440
	}
	vh := in.ViewportHeight
	if vh <= 0 {
		vh = 900
	}
	parent := in.ParentSize
	if parent <= 0 {
		parent = 16
	}

	// Convert to px first
	px, err := toPx(in.Value, in.From, base, vw, vh, parent)
	if err != nil {
		return errResult(err.Error())
	}
	// Convert from px to target
	result, err := fromPx(px, in.To, base, vw, vh, parent)
	if err != nil {
		return errResult(err.Error())
	}

	formatted := formatCSSValue(result, in.To)

	return resultJSON(map[string]any{
		"result":    result,
		"from":      in.From,
		"to":        in.To,
		"formatted": formatted,
	})
}

// toPx converts a value in the given unit to px.
func toPx(v float64, unit string, base, vw, vh, parent float64) (float64, error) {
	switch unit {
	case "px":
		return v, nil
	case "rem":
		return v * base, nil
	case "em":
		return v * parent, nil
	case "percent", "%":
		return v * parent / 100, nil
	case "vw":
		return v * vw / 100, nil
	case "vh":
		return v * vh / 100, nil
	default:
		return 0, fmt.Errorf("unknown unit %q: must be px, rem, em, percent, vw, or vh", unit)
	}
}

// fromPx converts a px value to the target unit.
func fromPx(px float64, unit string, base, vw, vh, parent float64) (float64, error) {
	switch unit {
	case "px":
		return px, nil
	case "rem":
		if base == 0 {
			return 0, fmt.Errorf("base_font_size cannot be zero for rem conversion")
		}
		return px / base, nil
	case "em":
		if parent == 0 {
			return 0, fmt.Errorf("parent_size cannot be zero for em conversion")
		}
		return px / parent, nil
	case "percent", "%":
		if parent == 0 {
			return 0, fmt.Errorf("parent_size cannot be zero for percent conversion")
		}
		return px * 100 / parent, nil
	case "vw":
		if vw == 0 {
			return 0, fmt.Errorf("viewport_width cannot be zero for vw conversion")
		}
		return px * 100 / vw, nil
	case "vh":
		if vh == 0 {
			return 0, fmt.Errorf("viewport_height cannot be zero for vh conversion")
		}
		return px * 100 / vh, nil
	default:
		return 0, fmt.Errorf("unknown unit %q: must be px, rem, em, percent, vw, or vh", unit)
	}
}

// formatCSSValue formats a float with unit suffix.
func formatCSSValue(v float64, unit string) string {
	suffix := unit
	if unit == "percent" {
		suffix = "%"
	}
	// Round to 4 decimal places to avoid floating-point noise.
	rounded := math.Round(v*10000) / 10000
	s := strconv.FormatFloat(rounded, 'f', -1, 64)
	return s + suffix
}

// ── frontend_breakpoint ───────────────────────────────────────────────────────

// BreakpointInput holds the parameters for the frontend_breakpoint tool.
type BreakpointInput struct {
	Width             int
	System            string         // tailwind | bootstrap | custom
	CustomBreakpoints map[string]int // key → min-width in px
	GenerateQuery     bool           // default true
}

// BreakpointResult is the JSON response for the frontend_breakpoint tool.
type BreakpointResult struct {
	Breakpoint string  `json:"breakpoint"`
	MinWidth   int     `json:"min_width"`
	MaxWidth   *int    `json:"max_width"`
	MediaQuery *string `json:"media_query,omitempty"`
}

// tailwindBreakpoints defines Tailwind v4 breakpoints (min-width, sorted).
var tailwindBreakpoints = []struct {
	name     string
	minWidth int
}{
	{"sm", 640},
	{"md", 768},
	{"lg", 1024},
	{"xl", 1280},
	{"2xl", 1536},
}

// bootstrapBreakpoints defines Bootstrap 5 breakpoints (min-width, sorted).
var bootstrapBreakpoints = []struct {
	name     string
	minWidth int
}{
	{"xs", 0},
	{"sm", 576},
	{"md", 768},
	{"lg", 992},
	{"xl", 1200},
	{"xxl", 1400},
}

// Breakpoint identifies the responsive breakpoint for a viewport width.
func Breakpoint(_ context.Context, in BreakpointInput) string {
	if in.Width < 0 {
		return errResult("width must be non-negative")
	}
	system := in.System
	if system == "" {
		system = "tailwind"
	}

	type bp struct {
		name     string
		minWidth int
	}
	var breakpoints []bp

	switch system {
	case "tailwind":
		// xs is an implicit 0-639 range with no Tailwind class
		breakpoints = []bp{{"xs", 0}}
		for _, b := range tailwindBreakpoints {
			breakpoints = append(breakpoints, bp{b.name, b.minWidth})
		}
	case "bootstrap":
		for _, b := range bootstrapBreakpoints {
			breakpoints = append(breakpoints, bp{b.name, b.minWidth})
		}
	case "custom":
		if len(in.CustomBreakpoints) == 0 {
			return errResult("custom_breakpoints must be provided when system is 'custom'")
		}
		for name, minW := range in.CustomBreakpoints {
			breakpoints = append(breakpoints, bp{name, minW})
		}
		// Sort by min-width
		sort.Slice(breakpoints, func(i, j int) bool {
			return breakpoints[i].minWidth < breakpoints[j].minWidth
		})
	default:
		return errResult("unknown system: must be tailwind, bootstrap, or custom")
	}

	// Find the matching breakpoint (largest min-width ≤ in.Width)
	matched := breakpoints[0]
	for _, b := range breakpoints {
		if in.Width >= b.minWidth {
			matched = b
		}
	}

	// Determine max-width (the next breakpoint's min-width - 1)
	var maxWidth *int
	for i, b := range breakpoints {
		if b.name == matched.name && i+1 < len(breakpoints) {
			next := breakpoints[i+1].minWidth - 1
			maxWidth = &next
			break
		}
	}

	res := BreakpointResult{
		Breakpoint: matched.name,
		MinWidth:   matched.minWidth,
		MaxWidth:   maxWidth,
	}

	if in.GenerateQuery {
		var query string
		if matched.minWidth == 0 {
			// For the smallest breakpoint, generate a max-width query if there's a next bp
			if maxWidth != nil {
				query = fmt.Sprintf("@media (max-width: %dpx) { ... }", *maxWidth)
			} else {
				query = "@media all { ... }"
			}
		} else {
			query = fmt.Sprintf("@media (min-width: %dpx) { ... }", matched.minWidth)
		}
		res.MediaQuery = &query
	}

	return resultJSON(res)
}

// ── frontend_regex ────────────────────────────────────────────────────────────

// RegexInput holds the parameters for the frontend_regex tool.
type RegexInput struct {
	Pattern     string
	Input       string
	Flags       string // i | m | g (combinable)
	Operation   string // test | match | replace, default "test"
	Replacement string // for replace operation
}

// RegexMatch represents a single match result.
type RegexMatch struct {
	Full   string   `json:"full"`
	Groups []string `json:"groups"`
	Index  int      `json:"index"`
}

// Regex tests, matches, or replaces using a Go regexp.
func Regex(_ context.Context, in RegexInput) string {
	if in.Pattern == "" {
		return errResult("pattern is required")
	}
	op := in.Operation
	if op == "" {
		op = "test"
	}

	// Build regexp pattern with flags
	prefix := buildFlagsPrefix(in.Flags)
	compiled, err := regexp.Compile(prefix + in.Pattern)
	if err != nil {
		return errResult("invalid regex pattern: " + err.Error())
	}

	switch op {
	case "test":
		return regexTest(compiled, in.Input)
	case "match":
		return regexMatch(compiled, in.Input)
	case "replace":
		return regexReplace(compiled, in.Input, in.Replacement, strings.Contains(in.Flags, "g"))
	default:
		return errResult("unknown operation: must be test, match, or replace")
	}
}

// buildFlagsPrefix converts flag characters to a Go regexp inline flag prefix.
func buildFlagsPrefix(flags string) string {
	var parts []string
	if strings.Contains(flags, "i") {
		parts = append(parts, "i")
	}
	if strings.Contains(flags, "m") {
		parts = append(parts, "m")
	}
	if strings.Contains(flags, "s") {
		parts = append(parts, "s")
	}
	if len(parts) == 0 {
		return ""
	}
	return "(?" + strings.Join(parts, "") + ")"
}

// regexTest returns {"matches": bool, "count": N}.
func regexTest(re *regexp.Regexp, input string) string {
	all := re.FindAllStringIndex(input, -1)
	count := len(all)
	matches := count > 0
	return resultJSON(map[string]any{
		"matches": matches,
		"count":   count,
	})
}

// regexMatch returns an array of match objects.
func regexMatch(re *regexp.Regexp, input string) string {
	allMatches := re.FindAllStringSubmatchIndex(input, -1)
	results := make([]RegexMatch, 0, len(allMatches))
	for _, loc := range allMatches {
		if len(loc) < 2 {
			continue
		}
		full := input[loc[0]:loc[1]]
		var groups []string
		// loc[2:] contains group pairs
		for i := 2; i < len(loc)-1; i += 2 {
			if loc[i] < 0 {
				groups = append(groups, "")
			} else {
				groups = append(groups, input[loc[i]:loc[i+1]])
			}
		}
		results = append(results, RegexMatch{
			Full:   full,
			Groups: groups,
			Index:  loc[0],
		})
	}
	return resultJSON(map[string]any{"matches": results})
}

// regexReplace replaces matches in input with replacement.
// If global is false, only the first match is replaced.
func regexReplace(re *regexp.Regexp, input, replacement string, global bool) string {
	var result string
	count := 0
	if global {
		allMatches := re.FindAllStringIndex(input, -1)
		count = len(allMatches)
		result = re.ReplaceAllString(input, replacement)
	} else {
		loc := re.FindStringIndex(input)
		if loc != nil {
			count = 1
		}
		result = re.ReplaceAllLiteralString(input, replacement)
		if count == 0 {
			result = input
		} else {
			result = input[:loc[0]] + replacement + input[loc[1]:]
		}
	}
	return resultJSON(map[string]any{
		"result": result,
		"count":  count,
	})
}

// ── frontend_locale_format ────────────────────────────────────────────────────

// LocaleFormatInput holds the parameters for the frontend_locale_format tool.
type LocaleFormatInput struct {
	Value    string         // number as string or date string
	Kind     string         // number | currency | date | time | datetime | percent
	Locale   string         // default "en-US"
	Currency string         // ISO 4217 code for currency kind
	Options  map[string]any // additional formatting options
}

// LocaleFormat formats numbers, dates, and currency using locale conventions.
func LocaleFormat(_ context.Context, in LocaleFormatInput) string {
	if in.Value == "" {
		return errResult("value is required")
	}
	if in.Kind == "" {
		return errResult("kind is required")
	}
	locale := in.Locale
	if locale == "" {
		locale = "en-US"
	}

	var formatted string
	var err error

	switch in.Kind {
	case "number":
		formatted, err = formatNumber(in.Value, locale, in.Options)
	case "currency":
		if in.Currency == "" {
			return errResult("currency code is required for kind=currency")
		}
		formatted, err = formatCurrency(in.Value, locale, in.Currency, in.Options)
	case "date":
		formatted, err = formatDate(in.Value, locale, "date", in.Options)
	case "time":
		formatted, err = formatDate(in.Value, locale, "time", in.Options)
	case "datetime":
		formatted, err = formatDate(in.Value, locale, "datetime", in.Options)
	case "percent":
		formatted, err = formatPercent(in.Value, locale, in.Options)
	default:
		return errResult("unknown kind: must be number, currency, date, time, datetime, or percent")
	}

	if err != nil {
		return errResult(err.Error())
	}

	return resultJSON(map[string]string{
		"formatted": formatted,
		"locale":    locale,
		"kind":      in.Kind,
	})
}

// localeConfig holds locale-specific formatting characters.
type localeConfig struct {
	thousandSep string
	decimalSep  string
	currencyPos string // before | after
}

// locales contains best-effort locale configurations for common locales.
var locales = map[string]localeConfig{
	"en-US": {thousandSep: ",", decimalSep: ".", currencyPos: "before"},
	"en-GB": {thousandSep: ",", decimalSep: ".", currencyPos: "before"},
	"de-DE": {thousandSep: ".", decimalSep: ",", currencyPos: "after"},
	"fr-FR": {thousandSep: "\u202f", decimalSep: ",", currencyPos: "after"},
	"es-ES": {thousandSep: ".", decimalSep: ",", currencyPos: "after"},
	"pt-BR": {thousandSep: ".", decimalSep: ",", currencyPos: "before"},
	"ja-JP": {thousandSep: ",", decimalSep: ".", currencyPos: "before"},
	"zh-CN": {thousandSep: ",", decimalSep: ".", currencyPos: "before"},
}

// getLocaleConfig returns the locale config, falling back to en-US.
func getLocaleConfig(locale string) localeConfig {
	if lc, ok := locales[locale]; ok {
		return lc
	}
	return locales["en-US"]
}

// formatNumber formats a numeric string according to locale conventions.
func formatNumber(value, locale string, opts map[string]any) (string, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number %q: %v", value, err)
	}
	lc := getLocaleConfig(locale)

	// Determine decimal places
	decimals := 2
	if dp, ok := opts["decimal_places"]; ok {
		switch v := dp.(type) {
		case float64:
			decimals = int(v)
		case int:
			decimals = v
		}
	}

	return applyLocaleFormat(f, lc, decimals), nil
}

// applyLocaleFormat formats a float with locale-specific separators.
func applyLocaleFormat(f float64, lc localeConfig, decimals int) string {
	negative := f < 0
	if negative {
		f = -f
	}

	// Format with the required decimal places
	formatted := strconv.FormatFloat(f, 'f', decimals, 64)

	// Split integer and decimal parts
	parts := strings.SplitN(formatted, ".", 2)
	intPart := parts[0]
	var decPart string
	if len(parts) > 1 {
		decPart = parts[1]
	}

	// Add thousand separators
	intPart = addThousandSeps(intPart, lc.thousandSep)

	var result string
	if decPart != "" {
		result = intPart + lc.decimalSep + decPart
	} else {
		result = intPart
	}

	if negative {
		result = "-" + result
	}
	return result
}

// addThousandSeps inserts thousand separators into an integer string.
func addThousandSeps(s, sep string) string {
	if sep == "" {
		return s
	}
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	start := n % 3
	b.WriteString(s[:start])
	for i := start; i < n; i += 3 {
		if i > 0 || start > 0 {
			b.WriteString(sep)
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// currencySymbols maps ISO 4217 currency codes to symbols.
var currencySymbols = map[string]string{
	"USD": "$", "EUR": "€", "GBP": "£", "JPY": "¥", "CNY": "¥",
	"CAD": "CA$", "AUD": "A$", "CHF": "Fr", "SEK": "kr", "NOK": "kr",
	"DKK": "kr", "BRL": "R$", "MXN": "MX$", "KRW": "₩", "INR": "₹",
	"RUB": "₽", "TRY": "₺", "PLN": "zł", "CZK": "Kč", "HUF": "Ft",
	"HKD": "HK$", "SGD": "S$", "NZD": "NZ$", "ZAR": "R",
}

// formatCurrency formats a number as a currency string.
func formatCurrency(value, locale, currency string, opts map[string]any) (string, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number %q: %v", value, err)
	}
	lc := getLocaleConfig(locale)

	// Japanese yen has 0 decimals by default
	decimals := 2
	if strings.ToUpper(currency) == "JPY" || strings.ToUpper(currency) == "KRW" {
		decimals = 0
	}
	if dp, ok := opts["decimal_places"]; ok {
		switch v := dp.(type) {
		case float64:
			decimals = int(v)
		case int:
			decimals = v
		}
	}

	sym := currency // fallback: use code if symbol not found
	if s, ok := currencySymbols[strings.ToUpper(currency)]; ok {
		sym = s
	}

	num := applyLocaleFormat(f, lc, decimals)
	if lc.currencyPos == "after" {
		return num + "\u00a0" + sym, nil
	}
	return sym + num, nil
}

// dateFormats maps locale + kind to Go time format strings.
// Best-effort for common locales.
var dateFormats = map[string]map[string]string{
	"date": {
		"en-US": "01/02/2006",
		"en-GB": "02/01/2006",
		"de-DE": "02.01.2006",
		"fr-FR": "02/01/2006",
		"es-ES": "02/01/2006",
		"pt-BR": "02/01/2006",
		"ja-JP": "2006/01/02",
		"zh-CN": "2006/01/02",
	},
	"time": {
		"en-US": "3:04 PM",
		"en-GB": "15:04",
		"de-DE": "15:04",
		"fr-FR": "15:04",
		"es-ES": "15:04",
		"pt-BR": "15:04",
		"ja-JP": "15:04",
		"zh-CN": "15:04",
	},
	"datetime": {
		"en-US": "01/02/2006, 3:04 PM",
		"en-GB": "02/01/2006 15:04",
		"de-DE": "02.01.2006 15:04",
		"fr-FR": "02/01/2006 15:04",
		"es-ES": "02/01/2006 15:04",
		"pt-BR": "02/01/2006 15:04",
		"ja-JP": "2006/01/02 15:04",
		"zh-CN": "2006/01/02 15:04",
	},
}

// parseDateTime attempts to parse a date string in common formats.
func parseDateTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"02.01.2006",
		"2006/01/02",
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822,
		time.RFC822Z,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as a date/time", value)
}

// formatDate parses and formats a date/time value for the given locale and kind.
func formatDate(value, locale, kind string, opts map[string]any) (string, error) {
	t, err := parseDateTime(value)
	if err != nil {
		return "", err
	}

	fmtMap, ok := dateFormats[kind]
	if !ok {
		return "", fmt.Errorf("unknown date kind: %s", kind)
	}

	layout, ok := fmtMap[locale]
	if !ok {
		layout = fmtMap["en-US"]
	}

	// Allow overriding layout via options
	if custom, ok := opts["format"]; ok {
		if s, ok := custom.(string); ok {
			layout = s
		}
	}

	return t.Format(layout), nil
}

// formatPercent formats a number as a percentage.
func formatPercent(value, locale string, opts map[string]any) (string, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number %q: %v", value, err)
	}
	lc := getLocaleConfig(locale)

	decimals := 1
	if dp, ok := opts["decimal_places"]; ok {
		switch v := dp.(type) {
		case float64:
			decimals = int(v)
		case int:
			decimals = v
		}
	}

	num := applyLocaleFormat(f*100, lc, decimals)
	return num + "%", nil
}

// ── frontend_icu_format ───────────────────────────────────────────────────────

// ICUFormatInput holds the parameters for the frontend_icu_format tool.
type ICUFormatInput struct {
	Template string
	Values   map[string]any
	Locale   string // default "en"
}

// ICUFormat evaluates a minimal ICU message format string.
// Supports: {variable}, {variable, plural, one{...} other{...}},
// {variable, select, key{...} other{...}}.
func ICUFormat(_ context.Context, in ICUFormatInput) string {
	if in.Template == "" {
		return errResult("template is required")
	}

	locale := in.Locale
	if locale == "" {
		locale = "en"
	}

	result, err := evaluateICU(in.Template, in.Values, locale)
	if err != nil {
		return errResult("icu format error: " + err.Error())
	}

	return resultJSON(map[string]string{"result": result})
}

// evaluateICU processes ICU message format template with variable bindings.
func evaluateICU(template string, values map[string]any, locale string) (string, error) {
	var sb strings.Builder
	i := 0
	runes := []rune(template)
	n := len(runes)

	for i < n {
		if runes[i] != '{' {
			sb.WriteRune(runes[i])
			i++
			continue
		}

		// Find matching closing brace (accounting for nesting)
		depth := 1
		j := i + 1
		for j < n && depth > 0 {
			if runes[j] == '{' {
				depth++
			} else if runes[j] == '}' {
				depth--
			}
			if depth > 0 {
				j++
			}
		}
		if depth != 0 {
			return "", fmt.Errorf("unmatched '{' at position %d", i)
		}

		// inner is the content between outermost { and }
		inner := string(runes[i+1 : j])
		expanded, err := processICUBlock(inner, values, locale)
		if err != nil {
			return "", err
		}
		sb.WriteString(expanded)
		i = j + 1
	}

	return sb.String(), nil
}

// processICUBlock processes the content of a single ICU block (without outer braces).
func processICUBlock(inner string, values map[string]any, locale string) (string, error) {
	// Split by comma to identify: varName, type, rest
	parts := strings.SplitN(inner, ",", 3)
	varName := strings.TrimSpace(parts[0])

	if len(parts) == 1 {
		// Simple variable substitution: {varName}
		val, ok := values[varName]
		if !ok {
			return "{" + varName + "}", nil
		}
		return fmt.Sprintf("%v", val), nil
	}

	blockType := strings.TrimSpace(parts[1])
	rest := ""
	if len(parts) == 3 {
		rest = strings.TrimSpace(parts[2])
	}

	val, _ := values[varName]

	switch blockType {
	case "plural":
		return processPluralBlock(val, rest, locale)
	case "select":
		return processSelectBlock(val, rest)
	default:
		// Unknown type — fall back to simple substitution
		return fmt.Sprintf("%v", val), nil
	}
}

// processPluralBlock handles plural ICU blocks.
// Format: one{...} other{...} (optionally: zero{...} two{...} few{...} many{...})
func processPluralBlock(val any, rest, locale string) (string, error) {
	n, err := toFloat(val)
	if err != nil {
		return "", fmt.Errorf("plural variable must be numeric, got %T", val)
	}

	clauses := parseICUClauses(rest)

	// Determine plural category
	category := pluralCategory(n, locale)

	// Try exact numeric match first (=0, =1, etc.)
	exact := "=" + formatICUNumber(n)
	if text, ok := clauses[exact]; ok {
		return evaluateICU(strings.ReplaceAll(text, "#", formatICUNumber(n)), val.(map[string]any), locale)
	}

	// Try category
	if text, ok := clauses[category]; ok {
		return evaluateICU(strings.ReplaceAll(text, "#", formatICUNumber(n)), nil, locale)
	}

	// Fallback to "other"
	if text, ok := clauses["other"]; ok {
		return evaluateICU(strings.ReplaceAll(text, "#", formatICUNumber(n)), nil, locale)
	}

	return fmt.Sprintf("%v", val), nil
}

// formatICUNumber formats a number for ICU substitution (integer if whole).
func formatICUNumber(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// pluralCategory returns the CLDR plural category for a number.
// This implements English rules (covers most Western locales).
func pluralCategory(n float64, locale string) string {
	// Simplified CLDR plural rules for supported locales.
	// Full CLDR would need a library; this covers the common cases.
	switch strings.ToLower(locale) {
	case "ja", "zh", "ko", "vi", "tr", "id":
		// These languages have no plural distinction
		return "other"
	}
	// English-like rule: n == 1 → "one", else "other"
	if n == 1 {
		return "one"
	}
	return "other"
}

// processSelectBlock handles select ICU blocks.
// Format: male{...} female{...} other{...}
func processSelectBlock(val any, rest string) (string, error) {
	key := fmt.Sprintf("%v", val)
	clauses := parseICUClauses(rest)

	if text, ok := clauses[key]; ok {
		return text, nil
	}
	if text, ok := clauses["other"]; ok {
		return text, nil
	}
	return key, nil
}

// parseICUClauses parses "key{text} key2{text2}" into a map.
func parseICUClauses(s string) map[string]string {
	clauses := make(map[string]string)
	runes := []rune(strings.TrimSpace(s))
	i := 0
	n := len(runes)

	for i < n {
		// Skip whitespace
		for i < n && unicode.IsSpace(runes[i]) {
			i++
		}
		if i >= n {
			break
		}

		// Read key (up to '{')
		keyStart := i
		for i < n && runes[i] != '{' {
			i++
		}
		if i >= n {
			break
		}
		key := strings.TrimSpace(string(runes[keyStart:i]))
		i++ // skip '{'

		// Read value (balanced braces)
		depth := 1
		valueStart := i
		for i < n && depth > 0 {
			if runes[i] == '{' {
				depth++
			} else if runes[i] == '}' {
				depth--
			}
			if depth > 0 {
				i++
			}
		}
		value := string(runes[valueStart:i])
		i++ // skip closing '}'

		if key != "" {
			clauses[key] = value
		}
	}

	return clauses
}

// toFloat converts various numeric types to float64.
func toFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", v)
	}
}
