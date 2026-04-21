package textenc_test

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/textenc"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// resultString unmarshals a single-key JSON response {"result": "..."} or {"slug": "..."} or {"value": "..."}.
func getString(t *testing.T, jsonStr, key string) string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonStr, err)
	}
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in %s", key, jsonStr)
	}
	return v
}

func getError(t *testing.T, jsonStr string) string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonStr, err)
	}
	return m["error"]
}

// ── text_escape ───────────────────────────────────────────────────────────────

func TestEscape(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.EscapeInput
		wantKey   string
		wantVal   string
		wantError bool
	}{
		{
			name:    "json escape newline",
			input:   textenc.EscapeInput{Text: "hello\nworld", Target: "json", Operation: "escape"},
			wantKey: "result", wantVal: `hello\nworld`,
		},
		{
			name:    "json unescape",
			input:   textenc.EscapeInput{Text: `hello\nworld`, Target: "json", Operation: "unescape"},
			wantKey: "result", wantVal: "hello\nworld",
		},
		{
			name:    "html escape",
			input:   textenc.EscapeInput{Text: "<b>bold & 'quoted'</b>", Target: "html", Operation: "escape"},
			wantKey: "result", wantVal: "&lt;b&gt;bold &amp; &#39;quoted&#39;&lt;/b&gt;",
		},
		{
			name:    "html unescape",
			input:   textenc.EscapeInput{Text: "&lt;b&gt;test&lt;/b&gt;", Target: "html", Operation: "unescape"},
			wantKey: "result", wantVal: "<b>test</b>",
		},
		{
			name:    "js escape",
			input:   textenc.EscapeInput{Text: `say "hello" & 'world'` + "\n", Target: "js", Operation: "escape"},
			wantKey: "result", wantVal: `say \"hello\" & \'world\'` + `\n`,
		},
		{
			name:    "js unescape",
			input:   textenc.EscapeInput{Text: `say \"hi\"`, Target: "js", Operation: "unescape"},
			wantKey: "result", wantVal: `say "hi"`,
		},
		{
			name:    "sql escape single quotes",
			input:   textenc.EscapeInput{Text: "it's a test", Target: "sql", Operation: "escape"},
			wantKey: "result", wantVal: "it''s a test",
		},
		{
			name:    "sql unescape single quotes",
			input:   textenc.EscapeInput{Text: "it''s a test", Target: "sql", Operation: "unescape"},
			wantKey: "result", wantVal: "it's a test",
		},
		{
			name:      "unknown target returns error",
			input:     textenc.EscapeInput{Text: "foo", Target: "xml", Operation: "escape"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.Escape(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, tc.wantKey)
			if val != tc.wantVal {
				t.Errorf("want %q, got %q", tc.wantVal, val)
			}
		})
	}
}

// ── text_slug ─────────────────────────────────────────────────────────────────

func TestSlug(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.SlugInput
		wantSlug  string
		wantError bool
	}{
		{
			name:     "basic ascii",
			input:    textenc.SlugInput{Text: "Hello World", Lower: true, Separator: "-"},
			wantSlug: "hello-world",
		},
		{
			name:     "accented characters",
			input:    textenc.SlugInput{Text: "Héllo Wörld", Lower: true, Separator: "-"},
			wantSlug: "hello-world",
		},
		{
			name:     "underscore separator",
			input:    textenc.SlugInput{Text: "hello world", Lower: true, Separator: "_"},
			wantSlug: "hello_world",
		},
		{
			name:     "no lower",
			input:    textenc.SlugInput{Text: "Hello World", Lower: false, Separator: "-"},
			wantSlug: "Hello-World",
		},
		{
			name:     "strips special chars",
			input:    textenc.SlugInput{Text: "hello@world.com", Lower: true, Separator: "-"},
			wantSlug: "hello-world-com",
		},
		{
			name:     "collapses multiple separators",
			input:    textenc.SlugInput{Text: "hello   world", Lower: true, Separator: "-"},
			wantSlug: "hello-world",
		},
		{
			name:      "empty text returns error",
			input:     textenc.SlugInput{Text: ""},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.Slug(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, "slug")
			if val != tc.wantSlug {
				t.Errorf("want %q, got %q", tc.wantSlug, val)
			}
		})
	}
}

// ── text_uuid ─────────────────────────────────────────────────────────────────

func TestUUID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.UUIDInput
		validate  func(t *testing.T, val string)
		wantError bool
	}{
		{
			name:  "uuid4 default",
			input: textenc.UUIDInput{Kind: "uuid4"},
			validate: func(t *testing.T, val string) {
				t.Helper()
				// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
				if len(val) != 36 {
					t.Errorf("expected 36 chars, got %d: %q", len(val), val)
				}
				if val[14] != '4' {
					t.Errorf("expected version 4, got char %q at pos 14", val[14])
				}
			},
		},
		{
			name:  "nanoid default length",
			input: textenc.UUIDInput{Kind: "nanoid", Length: 21},
			validate: func(t *testing.T, val string) {
				t.Helper()
				if len(val) != 21 {
					t.Errorf("expected 21 chars, got %d", len(val))
				}
			},
		},
		{
			name:  "token length 32 -> 64 hex chars",
			input: textenc.UUIDInput{Kind: "token", Length: 32},
			validate: func(t *testing.T, val string) {
				t.Helper()
				if len(val) != 64 {
					t.Errorf("expected 64 hex chars, got %d", len(val))
				}
			},
		},
		{
			name:  "ulid format",
			input: textenc.UUIDInput{Kind: "ulid"},
			validate: func(t *testing.T, val string) {
				t.Helper()
				if len(val) != 26 {
					t.Errorf("expected 26 chars for ulid, got %d", len(val))
				}
				if !regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`).MatchString(val) {
					t.Errorf("invalid ulid format: %q", val)
				}
			},
		},
		{
			name:      "unknown kind returns error",
			input:     textenc.UUIDInput{Kind: "snowflake"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.UUID(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, "value")
			if tc.validate != nil {
				tc.validate(t, val)
			}
		})
	}
}

func TestUUIDMultiple(t *testing.T) {
	ctx := context.Background()

	got := textenc.UUID(ctx, textenc.UUIDInput{Kind: "uuid4", Count: 3})
	var out map[string]any
	if err := json.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if e, ok := out["error"]; ok {
		t.Fatalf("unexpected error: %v", e)
	}

	values, ok := out["values"].([]any)
	if !ok {
		t.Fatalf("expected values array in response: %v", out)
	}
	if len(values) != 3 {
		t.Fatalf("expected 3 generated identifiers, got %d", len(values))
	}
}

// ── text_base64 ───────────────────────────────────────────────────────────────

func TestBase64(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.Base64Input
		wantVal   string
		wantError bool
	}{
		{
			name:    "standard encode",
			input:   textenc.Base64Input{Text: "hello world", Variant: "standard", Operation: "encode"},
			wantVal: "aGVsbG8gd29ybGQ=",
		},
		{
			name:    "standard decode",
			input:   textenc.Base64Input{Text: "aGVsbG8gd29ybGQ=", Variant: "standard", Operation: "decode"},
			wantVal: "hello world",
		},
		{
			name:    "urlsafe encode",
			input:   textenc.Base64Input{Text: "hello>world?", Variant: "urlsafe", Operation: "encode"},
			wantVal: "aGVsbG8-d29ybGQ_",
		},
		{
			name:    "urlsafe decode",
			input:   textenc.Base64Input{Text: "aGVsbG8-d29ybGQ_", Variant: "urlsafe", Operation: "decode"},
			wantVal: "hello>world?",
		},
		{
			name:      "unknown variant returns error",
			input:     textenc.Base64Input{Text: "foo", Variant: "base32", Operation: "encode"},
			wantError: true,
		},
		{
			name:      "invalid base64 decode returns error",
			input:     textenc.Base64Input{Text: "!!!notbase64!!!", Variant: "standard", Operation: "decode"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.Base64(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, "result")
			if val != tc.wantVal {
				t.Errorf("want %q, got %q", tc.wantVal, val)
			}
		})
	}
}

// ── text_url_encode ───────────────────────────────────────────────────────────

func TestURLEncode(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.URLEncodeInput
		wantVal   string
		wantError bool
	}{
		{
			name:    "query encode",
			input:   textenc.URLEncodeInput{Text: "hello world & foo=bar", Operation: "encode", Mode: "query"},
			wantVal: "hello+world+%26+foo%3Dbar",
		},
		{
			name:    "query decode",
			input:   textenc.URLEncodeInput{Text: "hello+world+%26+foo%3Dbar", Operation: "decode", Mode: "query"},
			wantVal: "hello world & foo=bar",
		},
		{
			name:    "path encode",
			input:   textenc.URLEncodeInput{Text: "my file name", Operation: "encode", Mode: "path"},
			wantVal: "my%20file%20name",
		},
		{
			name:    "path decode",
			input:   textenc.URLEncodeInput{Text: "my%20file%20name", Operation: "decode", Mode: "path"},
			wantVal: "my file name",
		},
		{
			name:      "unknown operation returns error",
			input:     textenc.URLEncodeInput{Text: "foo", Operation: "transform", Mode: "query"},
			wantError: true,
		},
		{
			name:      "unknown mode returns error",
			input:     textenc.URLEncodeInput{Text: "foo", Operation: "encode", Mode: "fragment"},
			wantError: true,
		},
		{
			name:      "invalid percent-encoding returns error",
			input:     textenc.URLEncodeInput{Text: "hello%ZZworld", Operation: "decode", Mode: "query"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.URLEncode(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, "result")
			if val != tc.wantVal {
				t.Errorf("want %q, got %q", tc.wantVal, val)
			}
		})
	}
}

// ── text_normalize ────────────────────────────────────────────────────────────

func TestNormalize(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.NormalizeInput
		wantVal   string
		wantError bool
	}{
		{
			name:    "trim whitespace",
			input:   textenc.NormalizeInput{Text: "  hello world  ", Operations: []string{"trim_whitespace"}},
			wantVal: "hello world",
		},
		{
			name:    "normalize newlines CRLF",
			input:   textenc.NormalizeInput{Text: "line1\r\nline2\r\nline3", Operations: []string{"normalize_newlines"}},
			wantVal: "line1\nline2\nline3",
		},
		{
			name:    "normalize newlines bare CR",
			input:   textenc.NormalizeInput{Text: "line1\rline2", Operations: []string{"normalize_newlines"}},
			wantVal: "line1\nline2",
		},
		{
			name:    "strip BOM",
			input:   textenc.NormalizeInput{Text: "\xef\xbb\xbfhello", Operations: []string{"strip_bom"}},
			wantVal: "hello",
		},
		{
			name:    "nfc normalization",
			input:   textenc.NormalizeInput{Text: "e\u0301", Operations: []string{"nfc"}}, // e + combining acute = é
			wantVal: "\u00e9",                                                             // precomposed é
		},
		{
			name:    "multiple operations",
			input:   textenc.NormalizeInput{Text: "  \r\nhello\r\n  ", Operations: []string{"normalize_newlines", "trim_whitespace"}},
			wantVal: "hello",
		},
		{
			name:      "empty operations returns error",
			input:     textenc.NormalizeInput{Text: "hello", Operations: []string{}},
			wantError: true,
		},
		{
			name:      "unknown operation returns error",
			input:     textenc.NormalizeInput{Text: "hello", Operations: []string{"uppercase"}},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.Normalize(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, "result")
			if val != tc.wantVal {
				t.Errorf("want %q, got %q", tc.wantVal, val)
			}
		})
	}
}

// ── text_case ─────────────────────────────────────────────────────────────────

func TestCase(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		input     textenc.CaseInput
		wantVal   string
		wantError bool
	}{
		{
			name:    "snake from spaces",
			input:   textenc.CaseInput{Text: "hello world foo", TargetCase: "snake"},
			wantVal: "hello_world_foo",
		},
		{
			name:    "kebab from spaces",
			input:   textenc.CaseInput{Text: "hello world foo", TargetCase: "kebab"},
			wantVal: "hello-world-foo",
		},
		{
			name:    "camel from snake",
			input:   textenc.CaseInput{Text: "hello_world_foo", TargetCase: "camel"},
			wantVal: "helloWorldFoo",
		},
		{
			name:    "pascal from snake",
			input:   textenc.CaseInput{Text: "hello_world_foo", TargetCase: "pascal"},
			wantVal: "HelloWorldFoo",
		},
		{
			name:    "screaming_snake from kebab",
			input:   textenc.CaseInput{Text: "hello-world-foo", TargetCase: "screaming_snake"},
			wantVal: "HELLO_WORLD_FOO",
		},
		{
			name:    "camel from camelCase input",
			input:   textenc.CaseInput{Text: "helloWorldFoo", TargetCase: "snake"},
			wantVal: "hello_world_foo",
		},
		{
			name:    "pascal from PascalCase input",
			input:   textenc.CaseInput{Text: "HelloWorldFoo", TargetCase: "kebab"},
			wantVal: "hello-world-foo",
		},
		{
			name:      "empty text returns error",
			input:     textenc.CaseInput{Text: "", TargetCase: "snake"},
			wantError: true,
		},
		{
			name:      "empty target_case returns error",
			input:     textenc.CaseInput{Text: "hello world", TargetCase: ""},
			wantError: true,
		},
		{
			name:      "unknown target_case returns error",
			input:     textenc.CaseInput{Text: "hello world", TargetCase: "title"},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := textenc.Case(ctx, tc.input)
			if tc.wantError {
				if getError(t, got) == "" {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			val := getString(t, got, "result")
			if val != tc.wantVal {
				t.Errorf("want %q, got %q", tc.wantVal, val)
			}
		})
	}
}

// ── uniqueness / no-collision smoke test ─────────────────────────────────────

func TestUUID_Uniqueness(t *testing.T) {
	ctx := context.Background()
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		got := textenc.UUID(ctx, textenc.UUIDInput{Kind: "uuid4"})
		val := getString(t, got, "value")
		if seen[val] {
			t.Fatalf("collision at iteration %d: %q", i, val)
		}
		seen[val] = true
	}
}

// ── round-trip test for base64 ────────────────────────────────────────────────

func TestBase64_RoundTrip(t *testing.T) {
	ctx := context.Background()
	original := "The quick brown fox jumps over the lazy dog 🦊"
	encoded := getString(t, textenc.Base64(ctx, textenc.Base64Input{Text: original, Variant: "standard", Operation: "encode"}), "result")
	decoded := getString(t, textenc.Base64(ctx, textenc.Base64Input{Text: encoded, Variant: "standard", Operation: "decode"}), "result")
	if decoded != original {
		t.Errorf("round-trip failed: want %q, got %q", original, decoded)
	}
}

// ── Normalize multiple operations chaining ───────────────────────────────────

func TestNormalize_MultipleOps(t *testing.T) {
	ctx := context.Background()
	// BOM + CRLF + trailing spaces
	input := "\xef\xbb\xbf  line1\r\nline2  "
	got := textenc.Normalize(ctx, textenc.NormalizeInput{
		Text:       input,
		Operations: []string{"strip_bom", "normalize_newlines", "trim_whitespace"},
	})
	val := getString(t, got, "result")
	want := "line1\nline2"
	if !strings.Contains(val, "line1") || !strings.Contains(val, "line2") {
		t.Errorf("unexpected result %q, want something containing %q", val, want)
	}
}

// ─── text_stats tests ─────────────────────────────────────────────────────────

func TestTextStats_Basic(t *testing.T) {
	ctx := context.Background()
	input := "This is a simple example text to test the word counter tool."
	result := textenc.TextStats(ctx, textenc.TextStatsInput{Text: input})
	var out textenc.TextStatsOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.WordCount != 12 {
		t.Errorf("expected word_count=12, got %d", out.WordCount)
	}
	if out.CharacterCount != 60 {
		t.Errorf("expected character_count=60, got %d", out.CharacterCount)
	}
	if out.CharacterCountNoSpaces != 49 {
		t.Errorf("expected character_count_no_spaces=49, got %d", out.CharacterCountNoSpaces)
	}
	if out.SentenceCount != 1 {
		t.Errorf("expected sentence_count=1, got %d", out.SentenceCount)
	}
	if out.UniqueWords != 12 {
		t.Errorf("expected unique_words=12, got %d", out.UniqueWords)
	}
	if out.Text != input {
		t.Errorf("expected text=%q, got %q", input, out.Text)
	}
}

func TestTextStats_MultipleSentences(t *testing.T) {
	ctx := context.Background()
	input := "Hello world! How are you today? I am fine."
	result := textenc.TextStats(ctx, textenc.TextStatsInput{Text: input})
	var out textenc.TextStatsOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.SentenceCount != 3 {
		t.Errorf("expected sentence_count=3, got %d", out.SentenceCount)
	}
	if out.WordCount != 9 {
		t.Errorf("expected word_count=9, got %d", out.WordCount)
	}
}

func TestTextStats_RepeatedWords(t *testing.T) {
	ctx := context.Background()
	input := "the cat sat on the mat the cat was fat"
	result := textenc.TextStats(ctx, textenc.TextStatsInput{Text: input})
	var out textenc.TextStatsOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.UniqueWords != 7 {
		t.Errorf("expected unique_words=7, got %d", out.UniqueWords)
	}
	if out.MostFrequentWord != "the" {
		t.Errorf("expected most_frequent_word='the', got %q", out.MostFrequentWord)
	}
}

func TestTextStats_MultipleParagraphs(t *testing.T) {
	ctx := context.Background()
	input := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."
	result := textenc.TextStats(ctx, textenc.TextStatsInput{Text: input})
	var out textenc.TextStatsOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	if out.ParagraphCount != 3 {
		t.Errorf("expected paragraph_count=3, got %d", out.ParagraphCount)
	}
}

func TestTextStats_AverageWordLength(t *testing.T) {
	ctx := context.Background()
	input := "a bb ccc dddd"
	result := textenc.TextStats(ctx, textenc.TextStatsInput{Text: input})
	var out textenc.TextStatsOutput
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("invalid JSON: %v — got: %s", err, result)
	}
	// (1 + 2 + 3 + 4) / 4 = 2.5
	if out.AverageWordLength != 2.5 {
		t.Errorf("expected average_word_length=2.5, got %f", out.AverageWordLength)
	}
}

func TestTextStats_EmptyText_ReturnsError(t *testing.T) {
	ctx := context.Background()
	result := textenc.TextStats(ctx, textenc.TextStatsInput{Text: ""})
	var errOut map[string]string
	if err := json.Unmarshal([]byte(result), &errOut); err != nil {
		t.Fatalf("expected error JSON, got: %s", result)
	}
	if _, ok := errOut["error"]; !ok {
		t.Error("expected error key in response")
	}
}
