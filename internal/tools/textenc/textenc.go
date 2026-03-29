// Package textenc implements MCP tools for text manipulation and encoding.
// All functions are stateless and safe for concurrent use.
package textenc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
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

// ── text_escape ──────────────────────────────────────────────────────────────

// EscapeInput holds the parameters for the text_escape tool.
type EscapeInput struct {
	Text      string
	Target    string // json | js | html | sql
	Operation string // escape | unescape
}

// Escape escapes or unescapes a string for the specified target syntax.
func Escape(_ context.Context, in EscapeInput) string {
	if in.Text == "" && in.Operation != "escape" {
		// Allow empty string through for escape; reject for unescape only when it's truly empty.
	}
	target := in.Target
	if target == "" {
		target = "json"
	}
	op := in.Operation
	if op == "" {
		op = "escape"
	}

	var result string
	switch target {
	case "json":
		result = escapeJSON(in.Text, op)
	case "html":
		if op == "escape" {
			result = html.EscapeString(in.Text)
		} else {
			result = html.UnescapeString(in.Text)
		}
	case "js":
		result = escapeJS(in.Text, op)
	case "sql":
		result = escapeSQL(in.Text, op)
	default:
		return errResult("unknown target: must be one of json, js, html, sql")
	}

	return resultJSON(map[string]string{"result": result})
}

// escapeJSON escapes or unescapes a string as if it were a JSON string value.
// For escape: marshal the string through encoding/json and strip surrounding quotes.
// For unescape: json.Unmarshal a quoted version of the input.
func escapeJSON(text, op string) string {
	if op == "escape" {
		b, err := json.Marshal(text)
		if err != nil {
			return text
		}
		// b is `"value"` — strip the outer quotes.
		return string(b[1 : len(b)-1])
	}
	// unescape: wrap in quotes and unmarshal
	var out string
	if err := json.Unmarshal([]byte(`"`+text+`"`), &out); err != nil {
		return text
	}
	return out
}

// escapeJS escapes/unescapes a string for JavaScript string literals.
// Escapes: \, ", ', \n, \r, \t.
func escapeJS(text, op string) string {
	if op == "escape" {
		r := strings.NewReplacer(
			`\`, `\\`,
			`"`, `\"`,
			`'`, `\'`,
			"\n", `\n`,
			"\r", `\r`,
			"\t", `\t`,
		)
		return r.Replace(text)
	}
	// unescape
	r := strings.NewReplacer(
		`\\`, `\`,
		`\"`, `"`,
		`\'`, `'`,
		`\n`, "\n",
		`\r`, "\r",
		`\t`, "\t",
	)
	return r.Replace(text)
}

// escapeSQL doubles single quotes for SQL string literals.
// There is no standard SQL unescape that differs from the escaped form,
// so unescape replaces doubled single-quotes with a single one.
func escapeSQL(text, op string) string {
	if op == "escape" {
		return strings.ReplaceAll(text, "'", "''")
	}
	return strings.ReplaceAll(text, "''", "'")
}

// ── text_slug ────────────────────────────────────────────────────────────────

// SlugInput holds the parameters for the text_slug tool.
type SlugInput struct {
	Text      string
	Separator string // default "-"
	Lower     bool   // default true
}

// latinMap maps common non-ASCII Latin characters to their ASCII approximation.
// This covers the most common accented letters used in Western European languages.
var latinMap = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a",
	'æ': "ae",
	'ç': "c",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i",
	'ñ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u",
	'ý': "y", 'ÿ': "y",
	'ß': "ss",
	'À': "a", 'Á': "a", 'Â': "a", 'Ã': "a", 'Ä': "a", 'Å': "a",
	'Æ': "ae",
	'Ç': "c",
	'È': "e", 'É': "e", 'Ê': "e", 'Ë': "e",
	'Ì': "i", 'Í': "i", 'Î': "i", 'Ï': "i",
	'Ñ': "n",
	'Ò': "o", 'Ó': "o", 'Ô': "o", 'Õ': "o", 'Ö': "o", 'Ø': "o",
	'Ù': "u", 'Ú': "u", 'Û': "u", 'Ü': "u",
	'Ý': "y",
}

// Slug converts arbitrary text to a URL-safe slug.
func Slug(_ context.Context, in SlugInput) string {
	if in.Text == "" {
		return errResult("text is required")
	}
	sep := in.Separator
	if sep == "" {
		sep = "-"
	}

	// 1. NFC normalize so combining characters are pre-composed.
	normalized := norm.NFC.String(in.Text)

	// 2. Map non-ASCII Latin characters to ASCII approximations.
	var sb strings.Builder
	for _, r := range normalized {
		if mapped, ok := latinMap[r]; ok {
			sb.WriteString(mapped)
		} else {
			sb.WriteRune(r)
		}
	}
	text := sb.String()

	// 3. Lowercase if requested (default).
	if in.Lower {
		text = strings.ToLower(text)
	}

	// 4. Replace spaces and hyphens/underscores with the separator.
	//    Strip all remaining non-alphanumeric characters.
	var out strings.Builder
	prevSep := false
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			prevSep = false
		} else if !prevSep {
			out.WriteString(sep)
			prevSep = true
		}
	}

	slug := strings.Trim(out.String(), sep)
	return resultJSON(map[string]string{"slug": slug})
}

// ── text_uuid ────────────────────────────────────────────────────────────────

// UUIDInput holds the parameters for the text_uuid tool.
type UUIDInput struct {
	Kind   string // uuid4 | nanoid | token
	Length int    // for nanoid and token (default 21)
}

// nanoidAlphabet is the URL-safe character set used by nanoid.
const nanoidAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"

// UUID generates a unique identifier of the requested kind.
func UUID(_ context.Context, in UUIDInput) string {
	kind := in.Kind
	if kind == "" {
		kind = "uuid4"
	}
	length := in.Length
	if length <= 0 {
		length = 21
	}

	switch kind {
	case "uuid4":
		id, err := uuid.NewRandom()
		if err != nil {
			return errResult("uuid generation failed: " + err.Error())
		}
		return resultJSON(map[string]string{"value": id.String()})

	case "nanoid":
		val, err := generateNanoid(length)
		if err != nil {
			return errResult("nanoid generation failed: " + err.Error())
		}
		return resultJSON(map[string]string{"value": val})

	case "token":
		// Generate hex-encoded random bytes. length bytes → 2*length hex chars.
		buf := make([]byte, length)
		if _, err := rand.Read(buf); err != nil {
			return errResult("token generation failed: " + err.Error())
		}
		return resultJSON(map[string]string{"value": hex.EncodeToString(buf)})

	default:
		return errResult("unknown kind: must be one of uuid4, nanoid, token")
	}
}

// generateNanoid produces a URL-safe random string of the given length
// using the nanoid alphabet (64 characters).
func generateNanoid(length int) (string, error) {
	alphabetLen := byte(len(nanoidAlphabet)) // 64
	mask := alphabetLen - 1                  // 0x3F — safe bitmask for power-of-two length

	buf := make([]byte, length*2) // over-allocate to reduce rand.Read calls
	result := make([]byte, length)
	generated := 0

	for generated < length {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			idx := b & mask
			result[generated] = nanoidAlphabet[idx]
			generated++
			if generated == length {
				break
			}
		}
	}
	return string(result), nil
}

// ── text_base64 ──────────────────────────────────────────────────────────────

// Base64Input holds the parameters for the text_base64 tool.
type Base64Input struct {
	Text      string
	Variant   string // standard | urlsafe
	Operation string // encode | decode
}

// Base64 encodes or decodes a string using the specified Base64 variant.
func Base64(_ context.Context, in Base64Input) string {
	if in.Operation == "decode" && in.Text == "" {
		return errResult("text is required for decode")
	}
	variant := in.Variant
	if variant == "" {
		variant = "standard"
	}
	op := in.Operation
	if op == "" {
		op = "encode"
	}

	var enc *base64.Encoding
	switch variant {
	case "standard":
		enc = base64.StdEncoding
	case "urlsafe":
		enc = base64.URLEncoding
	default:
		return errResult("unknown variant: must be standard or urlsafe")
	}

	switch op {
	case "encode":
		return resultJSON(map[string]string{"result": enc.EncodeToString([]byte(in.Text))})
	case "decode":
		decoded, err := enc.DecodeString(in.Text)
		if err != nil {
			// Also try without padding for convenience.
			decoded, err = enc.WithPadding(base64.NoPadding).DecodeString(in.Text)
			if err != nil {
				return errResult("base64 decode failed: " + err.Error())
			}
		}
		return resultJSON(map[string]string{"result": string(decoded)})
	default:
		return errResult("unknown operation: must be encode or decode")
	}
}

// ── text_url_encode ──────────────────────────────────────────────────────────

// URLEncodeInput holds the parameters for the text_url_encode tool.
type URLEncodeInput struct {
	Text      string
	Operation string // encode | decode
	Mode      string // query | path
}

// URLEncode percent-encodes or decodes a URL component.
func URLEncode(_ context.Context, in URLEncodeInput) string {
	if in.Text == "" && in.Operation != "encode" {
		return errResult("text is required")
	}
	op := in.Operation
	if op == "" {
		op = "encode"
	}
	mode := in.Mode
	if mode == "" {
		mode = "query"
	}

	switch op {
	case "encode":
		switch mode {
		case "query":
			return resultJSON(map[string]string{"result": url.QueryEscape(in.Text)})
		case "path":
			return resultJSON(map[string]string{"result": url.PathEscape(in.Text)})
		default:
			return errResult("unknown mode: must be query or path")
		}
	case "decode":
		switch mode {
		case "query":
			decoded, err := url.QueryUnescape(in.Text)
			if err != nil {
				return errResult("query decode failed: " + err.Error())
			}
			return resultJSON(map[string]string{"result": decoded})
		case "path":
			decoded, err := url.PathUnescape(in.Text)
			if err != nil {
				return errResult("path decode failed: " + err.Error())
			}
			return resultJSON(map[string]string{"result": decoded})
		default:
			return errResult("unknown mode: must be query or path")
		}
	default:
		return errResult("unknown operation: must be encode or decode")
	}
}

// ── text_normalize ───────────────────────────────────────────────────────────

// NormalizeInput holds the parameters for the text_normalize tool.
type NormalizeInput struct {
	Text       string
	Operations []string // trim_whitespace | normalize_newlines | strip_bom | nfc | nfd | nfkc | nfkd
}

// utf8BOM is the UTF-8 byte order mark.
const utf8BOM = "\xef\xbb\xbf"

// Normalize applies a sequence of normalization operations to the given text.
func Normalize(_ context.Context, in NormalizeInput) string {
	if len(in.Operations) == 0 {
		return errResult("at least one operation is required")
	}

	text := in.Text
	for _, op := range in.Operations {
		switch op {
		case "trim_whitespace":
			text = strings.TrimSpace(text)
		case "normalize_newlines":
			// Normalize \r\n and bare \r to \n.
			text = strings.ReplaceAll(text, "\r\n", "\n")
			text = strings.ReplaceAll(text, "\r", "\n")
		case "strip_bom":
			text = strings.TrimPrefix(text, utf8BOM)
		case "nfc":
			text = norm.NFC.String(text)
		case "nfd":
			text = norm.NFD.String(text)
		case "nfkc":
			text = norm.NFKC.String(text)
		case "nfkd":
			text = norm.NFKD.String(text)
		default:
			return errResult("unknown operation: " + op + "; valid values are trim_whitespace, normalize_newlines, strip_bom, nfc, nfd, nfkc, nfkd")
		}
	}

	return resultJSON(map[string]string{"result": text})
}

// ── text_case ────────────────────────────────────────────────────────────────

// CaseInput holds the parameters for the text_case tool.
type CaseInput struct {
	Text       string
	TargetCase string // camel | snake | kebab | pascal | screaming_snake
}

// wordBoundaryRe matches uppercase letters following lowercase letters (camelCase boundary)
// or any non-alphanumeric run used as a word separator.
var wordBoundaryRe = regexp.MustCompile(`[_\-\s]+|([a-z])([A-Z])`)

// tokenize splits text into lowercase word tokens by detecting word boundaries:
// spaces, underscores, hyphens, and camelCase/PascalCase transitions.
func tokenize(text string) []string {
	// Insert a space before each uppercase letter that follows a lowercase letter
	// so that camelCase and PascalCase boundaries are treated as separators.
	var expanded strings.Builder
	runes := []rune(text)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && unicode.IsLower(runes[i-1]) {
			expanded.WriteRune(' ')
		}
		expanded.WriteRune(r)
	}

	// Split on spaces, underscores, and hyphens; filter empty strings.
	parts := regexp.MustCompile(`[\s_\-]+`).Split(expanded.String(), -1)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			tokens = append(tokens, strings.ToLower(p))
		}
	}
	return tokens
}

// Case converts text between common naming conventions.
func Case(_ context.Context, in CaseInput) string {
	if in.Text == "" {
		return errResult("text is required")
	}
	if in.TargetCase == "" {
		return errResult("target_case is required")
	}

	tokens := tokenize(in.Text)
	if len(tokens) == 0 {
		return resultJSON(map[string]string{"result": ""})
	}

	var result string
	switch in.TargetCase {
	case "snake":
		result = strings.Join(tokens, "_")
	case "kebab":
		result = strings.Join(tokens, "-")
	case "screaming_snake":
		upper := make([]string, len(tokens))
		for i, t := range tokens {
			upper[i] = strings.ToUpper(t)
		}
		result = strings.Join(upper, "_")
	case "camel":
		var sb strings.Builder
		for i, t := range tokens {
			if i == 0 {
				sb.WriteString(t)
			} else {
				sb.WriteString(capitalize(t))
			}
		}
		result = sb.String()
	case "pascal":
		var sb strings.Builder
		for _, t := range tokens {
			sb.WriteString(capitalize(t))
		}
		result = sb.String()
	default:
		return errResult("unknown target_case: must be one of camel, snake, kebab, pascal, screaming_snake")
	}

	return resultJSON(map[string]string{"result": result})
}

// capitalize returns the string with its first rune uppercased.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
