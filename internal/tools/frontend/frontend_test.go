package frontend_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/frontend"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func isErrorJSON(s string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return false
	}
	_, ok := m["error"]
	return ok
}

func getStringField(t *testing.T, s, field string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("unmarshal failed: %v — raw: %s", err, s)
	}
	v, ok := m[field]
	if !ok {
		t.Fatalf("field %q not found in %s", field, s)
	}
	str, ok := v.(string)
	if !ok {
		t.Fatalf("field %q is not a string in %s", field, s)
	}
	return str
}

func getFloat64Field(t *testing.T, s, field string) float64 {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("unmarshal failed: %v — raw: %s", err, s)
	}
	v, ok := m[field]
	if !ok {
		t.Fatalf("field %q not found in %s", field, s)
	}
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("field %q is not a float64 in %s", field, s)
	}
	return f
}

func getBoolField(t *testing.T, s, field string) bool {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("unmarshal failed: %v — raw: %s", err, s)
	}
	v, ok := m[field]
	if !ok {
		t.Fatalf("field %q not found in %s", field, s)
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("field %q is not a bool in %s", field, s)
	}
	return b
}

// ── frontend_color ────────────────────────────────────────────────────────────

func TestColor(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     frontend.ColorInput
		wantErr   bool
		wantField string
		wantValue string
	}{
		{
			name:      "hex to rgb happy path",
			input:     frontend.ColorInput{Color: "#ff0000", To: "rgb"},
			wantErr:   false,
			wantField: "result",
			wantValue: "rgb(255, 0, 0)",
		},
		{
			name:      "hex to hsl",
			input:     frontend.ColorInput{Color: "#ff0000", To: "hsl"},
			wantErr:   false,
			wantField: "result",
			wantValue: "hsl(0, 100%, 50%)",
		},
		{
			name:      "rgb to hex",
			input:     frontend.ColorInput{Color: "rgb(0, 128, 0)", To: "hex"},
			wantErr:   false,
			wantField: "result",
			wantValue: "#008000",
		},
		{
			name:      "hsl to hex",
			input:     frontend.ColorInput{Color: "hsl(240, 100%, 50%)", To: "hex"},
			wantErr:   false,
			wantField: "result",
			wantValue: "#0000ff",
		},
		{
			name:      "hex to rgba with alpha",
			input:     frontend.ColorInput{Color: "#000000", To: "rgba", Alpha: 0.5},
			wantErr:   false,
			wantField: "result",
			wantValue: "rgba(0, 0, 0, 0.50)",
		},
		{
			name:    "missing color returns error",
			input:   frontend.ColorInput{},
			wantErr: true,
		},
		{
			name:    "invalid color format returns error",
			input:   frontend.ColorInput{Color: "notacolor"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frontend.Color(ctx, tc.input)
			if tc.wantErr {
				if !isErrorJSON(got) {
					t.Errorf("expected error JSON, got: %s", got)
				}
				return
			}
			if isErrorJSON(got) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.wantField != "" {
				val := getStringField(t, got, tc.wantField)
				if val != tc.wantValue {
					t.Errorf("got %q, want %q", val, tc.wantValue)
				}
			}
		})
	}
}

func TestColorContrast(t *testing.T) {
	ctx := context.Background()

	// Black on white: contrast ratio ≈ 21:1 — should be WCAG AAA
	result := frontend.Color(ctx, frontend.ColorInput{
		Color:   "#000000",
		To:      "hex",
		Against: "#ffffff",
	})
	if isErrorJSON(result) {
		t.Fatalf("unexpected error: %s", result)
	}

	ratio := getFloat64Field(t, result, "contrast_ratio")
	if ratio < 20 || ratio > 22 {
		t.Errorf("expected contrast ratio ~21, got %f", ratio)
	}

	var m map[string]any
	json.Unmarshal([]byte(result), &m)
	aa := m["wcag_aa"].(bool)
	aaa := m["wcag_aaa"].(bool)
	if !aa || !aaa {
		t.Errorf("expected WCAG AA and AAA to be true for black on white, got aa=%v aaa=%v", aa, aaa)
	}
}

func TestColorContrastFail(t *testing.T) {
	ctx := context.Background()

	// White on white: contrast ratio = 1:1 — should fail both
	result := frontend.Color(ctx, frontend.ColorInput{
		Color:   "#ffffff",
		To:      "hex",
		Against: "#ffffff",
	})
	if isErrorJSON(result) {
		t.Fatalf("unexpected error: %s", result)
	}

	ratio := getFloat64Field(t, result, "contrast_ratio")
	if ratio != 1.0 {
		t.Errorf("expected contrast ratio 1.0, got %f", ratio)
	}
	aa := getBoolField(t, result, "wcag_aa")
	aaa := getBoolField(t, result, "wcag_aaa")
	if aa || aaa {
		t.Errorf("expected WCAG AA and AAA to be false for white on white")
	}
}

// ── frontend_css_unit ─────────────────────────────────────────────────────────

func TestCSSUnit(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   frontend.CSSUnitInput
		wantErr bool
		wantVal float64
	}{
		{
			name:    "px to rem happy path",
			input:   frontend.CSSUnitInput{Value: 16, From: "px", To: "rem", BaseFontSize: 16},
			wantErr: false,
			wantVal: 1.0,
		},
		{
			name:    "rem to px",
			input:   frontend.CSSUnitInput{Value: 1.5, From: "rem", To: "px", BaseFontSize: 16},
			wantErr: false,
			wantVal: 24,
		},
		{
			name:    "px to vw",
			input:   frontend.CSSUnitInput{Value: 1440, From: "px", To: "vw", ViewportWidth: 1440},
			wantErr: false,
			wantVal: 100,
		},
		{
			name:    "em to px",
			input:   frontend.CSSUnitInput{Value: 2, From: "em", To: "px", ParentSize: 16},
			wantErr: false,
			wantVal: 32,
		},
		{
			name:    "percent to px",
			input:   frontend.CSSUnitInput{Value: 50, From: "percent", To: "px", ParentSize: 200},
			wantErr: false,
			wantVal: 100,
		},
		{
			name:    "missing from returns error",
			input:   frontend.CSSUnitInput{Value: 16, To: "rem"},
			wantErr: true,
		},
		{
			name:    "invalid unit returns error",
			input:   frontend.CSSUnitInput{Value: 16, From: "px", To: "invalid"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frontend.CSSUnit(ctx, tc.input)
			if tc.wantErr {
				if !isErrorJSON(got) {
					t.Errorf("expected error JSON, got: %s", got)
				}
				return
			}
			if isErrorJSON(got) {
				t.Fatalf("unexpected error: %s", got)
			}
			val := getFloat64Field(t, got, "result")
			if val != tc.wantVal {
				t.Errorf("got %f, want %f", val, tc.wantVal)
			}
		})
	}
}

// ── frontend_breakpoint ───────────────────────────────────────────────────────

func TestBreakpoint(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     frontend.BreakpointInput
		wantErr   bool
		wantBP    string
		wantQuery string
	}{
		{
			name:      "tailwind sm at 768",
			input:     frontend.BreakpointInput{Width: 768, System: "tailwind", GenerateQuery: true},
			wantErr:   false,
			wantBP:    "md",
			wantQuery: "@media (min-width: 768px) { ... }",
		},
		{
			name:    "tailwind xs at 320",
			input:   frontend.BreakpointInput{Width: 320, System: "tailwind"},
			wantErr: false,
			wantBP:  "xs",
		},
		{
			name:    "tailwind 2xl at 1600",
			input:   frontend.BreakpointInput{Width: 1600, System: "tailwind"},
			wantErr: false,
			wantBP:  "2xl",
		},
		{
			name:    "bootstrap md at 800",
			input:   frontend.BreakpointInput{Width: 800, System: "bootstrap"},
			wantErr: false,
			wantBP:  "md",
		},
		{
			name:   "custom breakpoints",
			input:  frontend.BreakpointInput{Width: 900, System: "custom", CustomBreakpoints: map[string]int{"mobile": 0, "tablet": 600, "desktop": 1024}},
			wantBP: "tablet",
		},
		{
			name:    "unknown system returns error",
			input:   frontend.BreakpointInput{Width: 800, System: "unknown"},
			wantErr: true,
		},
		{
			name:    "custom system without breakpoints returns error",
			input:   frontend.BreakpointInput{Width: 800, System: "custom"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frontend.Breakpoint(ctx, tc.input)
			if tc.wantErr {
				if !isErrorJSON(got) {
					t.Errorf("expected error JSON, got: %s", got)
				}
				return
			}
			if isErrorJSON(got) {
				t.Fatalf("unexpected error: %s", got)
			}
			bp := getStringField(t, got, "breakpoint")
			if bp != tc.wantBP {
				t.Errorf("got breakpoint %q, want %q", bp, tc.wantBP)
			}
			if tc.wantQuery != "" {
				var m map[string]any
				json.Unmarshal([]byte(got), &m)
				query, _ := m["media_query"].(string)
				if query != tc.wantQuery {
					t.Errorf("got media_query %q, want %q", query, tc.wantQuery)
				}
			}
		})
	}
}

// ── regex_test ──────────────────────────────────────────────────────────────

func TestRegex(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   frontend.RegexInput
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		{
			name:    "test operation happy path",
			input:   frontend.RegexInput{Pattern: `\d+`, Input: "hello 123 world", Operation: "test"},
			wantErr: false,
			check: func(t *testing.T, result string) {
				t.Helper()
				var m map[string]any
				json.Unmarshal([]byte(result), &m)
				if m["matches"] != true {
					t.Errorf("expected matches=true, got: %s", result)
				}
				if m["count"].(float64) != 1 {
					t.Errorf("expected count=1, got: %s", result)
				}
			},
		},
		{
			name:    "test no match",
			input:   frontend.RegexInput{Pattern: `\d+`, Input: "hello world", Operation: "test"},
			wantErr: false,
			check: func(t *testing.T, result string) {
				t.Helper()
				var m map[string]any
				json.Unmarshal([]byte(result), &m)
				if m["matches"] != false {
					t.Errorf("expected matches=false, got: %s", result)
				}
			},
		},
		{
			name:    "match operation with groups",
			input:   frontend.RegexInput{Pattern: `(\w+)@(\w+)`, Input: "user@example.com", Operation: "match"},
			wantErr: false,
			check: func(t *testing.T, result string) {
				t.Helper()
				var m map[string]any
				json.Unmarshal([]byte(result), &m)
				matches := m["matches"].([]any)
				if len(matches) != 1 {
					t.Errorf("expected 1 match, got %d", len(matches))
				}
				first := matches[0].(map[string]any)
				if first["full"] != "user@example" {
					t.Errorf("expected full=user@example, got %s", first["full"])
				}
			},
		},
		{
			name:    "replace operation",
			input:   frontend.RegexInput{Pattern: `\d+`, Input: "price: 100 USD", Operation: "replace", Replacement: "XXX", Flags: "g"},
			wantErr: false,
			check: func(t *testing.T, result string) {
				t.Helper()
				got := getStringField(t, result, "result")
				if got != "price: XXX USD" {
					t.Errorf("got %q, want %q", got, "price: XXX USD")
				}
			},
		},
		{
			name:    "case-insensitive flag",
			input:   frontend.RegexInput{Pattern: `hello`, Input: "Hello World", Operation: "test", Flags: "i"},
			wantErr: false,
			check: func(t *testing.T, result string) {
				t.Helper()
				var m map[string]any
				json.Unmarshal([]byte(result), &m)
				if m["matches"] != true {
					t.Errorf("expected case-insensitive match, got: %s", result)
				}
			},
		},
		{
			name:    "missing pattern returns error",
			input:   frontend.RegexInput{Input: "hello"},
			wantErr: true,
		},
		{
			name:    "invalid regex returns error",
			input:   frontend.RegexInput{Pattern: `[invalid`, Input: "hello"},
			wantErr: true,
		},
		{
			name:    "unknown operation returns error",
			input:   frontend.RegexInput{Pattern: `\d+`, Input: "123", Operation: "unknown"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frontend.Regex(ctx, tc.input)
			if tc.wantErr {
				if !isErrorJSON(got) {
					t.Errorf("expected error JSON, got: %s", got)
				}
				return
			}
			if isErrorJSON(got) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

// ── frontend_locale_format ────────────────────────────────────────────────────

func TestLocaleFormat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     frontend.LocaleFormatInput
		wantErr   bool
		wantField string
		contains  string
	}{
		{
			name:      "number en-US happy path",
			input:     frontend.LocaleFormatInput{Value: "1234567.89", Kind: "number", Locale: "en-US"},
			wantField: "formatted",
			contains:  "1,234,567",
		},
		{
			name:      "number de-DE uses dot separator",
			input:     frontend.LocaleFormatInput{Value: "1234567.89", Kind: "number", Locale: "de-DE"},
			wantField: "formatted",
			contains:  "1.234.567",
		},
		{
			name:      "currency USD en-US",
			input:     frontend.LocaleFormatInput{Value: "1234.56", Kind: "currency", Locale: "en-US", Currency: "USD"},
			wantField: "formatted",
			contains:  "$1,234",
		},
		{
			name:      "currency EUR de-DE",
			input:     frontend.LocaleFormatInput{Value: "1234.56", Kind: "currency", Locale: "de-DE", Currency: "EUR"},
			wantField: "formatted",
			contains:  "€",
		},
		{
			name:      "date en-US",
			input:     frontend.LocaleFormatInput{Value: "2024-03-15", Kind: "date", Locale: "en-US"},
			wantField: "formatted",
			contains:  "03/15/2024",
		},
		{
			name:      "date de-DE",
			input:     frontend.LocaleFormatInput{Value: "2024-03-15", Kind: "date", Locale: "de-DE"},
			wantField: "formatted",
			contains:  "15.03.2024",
		},
		{
			name:      "percent",
			input:     frontend.LocaleFormatInput{Value: "0.85", Kind: "percent", Locale: "en-US"},
			wantField: "formatted",
			contains:  "85",
		},
		{
			name:    "missing value returns error",
			input:   frontend.LocaleFormatInput{Kind: "number"},
			wantErr: true,
		},
		{
			name:    "missing kind returns error",
			input:   frontend.LocaleFormatInput{Value: "123"},
			wantErr: true,
		},
		{
			name:    "currency without code returns error",
			input:   frontend.LocaleFormatInput{Value: "123", Kind: "currency", Locale: "en-US"},
			wantErr: true,
		},
		{
			name:    "invalid number returns error",
			input:   frontend.LocaleFormatInput{Value: "notanumber", Kind: "number"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frontend.LocaleFormat(ctx, tc.input)
			if tc.wantErr {
				if !isErrorJSON(got) {
					t.Errorf("expected error JSON, got: %s", got)
				}
				return
			}
			if isErrorJSON(got) {
				t.Fatalf("unexpected error: %s", got)
			}
			if tc.wantField != "" {
				val := getStringField(t, got, tc.wantField)
				if tc.contains != "" && !strings.Contains(val, tc.contains) {
					t.Errorf("got %q, expected to contain %q", val, tc.contains)
				}
			}
		})
	}
}

// ── frontend_icu_format ───────────────────────────────────────────────────────

func TestICUFormat(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		input   frontend.ICUFormatInput
		wantErr bool
		want    string
	}{
		{
			name: "simple variable substitution",
			input: frontend.ICUFormatInput{
				Template: "Hello, {name}!",
				Values:   map[string]any{"name": "World"},
			},
			want: "Hello, World!",
		},
		{
			name: "plural one",
			input: frontend.ICUFormatInput{
				Template: "You have {count, plural, one{# message} other{# messages}}",
				Values:   map[string]any{"count": 1.0},
				Locale:   "en",
			},
			want: "You have 1 message",
		},
		{
			name: "plural other",
			input: frontend.ICUFormatInput{
				Template: "You have {count, plural, one{# message} other{# messages}}",
				Values:   map[string]any{"count": 5.0},
				Locale:   "en",
			},
			want: "You have 5 messages",
		},
		{
			name: "select male",
			input: frontend.ICUFormatInput{
				Template: "{gender, select, male{He} female{She} other{They}} likes this.",
				Values:   map[string]any{"gender": "male"},
			},
			want: "He likes this.",
		},
		{
			name: "select female",
			input: frontend.ICUFormatInput{
				Template: "{gender, select, male{He} female{She} other{They}} likes this.",
				Values:   map[string]any{"gender": "female"},
			},
			want: "She likes this.",
		},
		{
			name: "select other fallback",
			input: frontend.ICUFormatInput{
				Template: "{gender, select, male{He} female{She} other{They}} likes this.",
				Values:   map[string]any{"gender": "nonbinary"},
			},
			want: "They likes this.",
		},
		{
			name: "missing variable keeps placeholder",
			input: frontend.ICUFormatInput{
				Template: "Hello, {name}!",
				Values:   map[string]any{},
			},
			want: "Hello, {name}!",
		},
		{
			name:    "missing template returns error",
			input:   frontend.ICUFormatInput{Values: map[string]any{"name": "World"}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frontend.ICUFormat(ctx, tc.input)
			if tc.wantErr {
				if !isErrorJSON(got) {
					t.Errorf("expected error JSON, got: %s", got)
				}
				return
			}
			if isErrorJSON(got) {
				t.Fatalf("unexpected error: %s", got)
			}
			result := getStringField(t, got, "result")
			if result != tc.want {
				t.Errorf("got %q, want %q", result, tc.want)
			}
		})
	}
}
