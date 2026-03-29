// Package httptools implements MCP tools for HTTP and networking operations.
// Provides http_request, http_curl_convert, http_webhook_replay,
// http_signed_url, and http_url_parse.
package httptools

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodyBytes is the maximum response body size we read (1 MB).
const maxBodyBytes = 1 << 20

// errResult returns a JSON-encoded error response.
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

// ─── http_request ──────────────────────────────────────────────────────────

// HTTPRequestInput is the input for the http_request tool.
type HTTPRequestInput struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
	TimeoutSeconds  int               `json:"timeout_seconds"`
	FollowRedirects bool              `json:"follow_redirects"`
}

// HTTPRequestOutput is the output for the http_request tool.
type HTTPRequestOutput struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	DurationMS int64             `json:"duration_ms"`
}

// HTTPRequest performs an HTTP request and returns status, headers, body, and duration.
func HTTPRequest(ctx context.Context, input HTTPRequestInput) string {
	if strings.TrimSpace(input.URL) == "" {
		return errResult("url is required")
	}

	method := strings.ToUpper(input.Method)
	if method == "" {
		method = "GET"
	}

	timeoutSecs := input.TimeoutSeconds
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSecs) * time.Second,
	}
	if !input.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var bodyReader io.Reader
	if input.Body != "" {
		bodyReader = strings.NewReader(input.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, input.URL, bodyReader)
	if err != nil {
		return errResult("failed to create request: " + err.Error())
	}

	for k, v := range input.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return errResult("request failed: " + err.Error())
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, maxBodyBytes)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return errResult("failed to read response body: " + err.Error())
	}

	headers := make(map[string]string, len(resp.Header))
	for k, vals := range resp.Header {
		headers[k] = strings.Join(vals, ", ")
	}

	return resultJSON(HTTPRequestOutput{
		Status:     resp.StatusCode,
		Headers:    headers,
		Body:       string(bodyBytes),
		DurationMS: elapsed.Milliseconds(),
	})
}

// ─── http_curl_convert ─────────────────────────────────────────────────────

// HTTPCurlConvertInput is the input for the http_curl_convert tool.
type HTTPCurlConvertInput struct {
	Curl   string `json:"curl"`
	Target string `json:"target"`
}

// HTTPCurlConvertOutput is the output for the http_curl_convert tool.
type HTTPCurlConvertOutput struct {
	Snippet string `json:"snippet"`
	Target  string `json:"target"`
}

// parsedCurl holds fields extracted from a curl command string.
type parsedCurl struct {
	url     string
	method  string
	headers map[string]string
	body    string
}

// parseCurl extracts URL, method, headers, and body from a curl command string.
// Only handles common flags: -X, -H, -d/--data, --url, positional URL.
func parseCurl(cmd string) parsedCurl {
	result := parsedCurl{
		method:  "GET",
		headers: make(map[string]string),
	}

	// tokenize respecting quoted strings
	tokens := tokenizeCurl(cmd)

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "curl":
			// skip
		case tok == "-X" || tok == "--request":
			if i+1 < len(tokens) {
				i++
				result.method = strings.ToUpper(tokens[i])
			}
		case tok == "-H" || tok == "--header":
			if i+1 < len(tokens) {
				i++
				k, v, found := strings.Cut(tokens[i], ":")
				if found {
					result.headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
				}
			}
		case tok == "-d" || tok == "--data" || tok == "--data-raw" || tok == "--data-binary":
			if i+1 < len(tokens) {
				i++
				result.body = tokens[i]
				if result.method == "GET" {
					result.method = "POST"
				}
			}
		case tok == "--url":
			if i+1 < len(tokens) {
				i++
				result.url = tokens[i]
			}
		case !strings.HasPrefix(tok, "-") && result.url == "":
			// bare positional argument — treat as URL
			result.url = tok
		}
	}

	return result
}

// tokenizeCurl splits a curl command string into tokens, respecting single and
// double quoted strings and backslash-escaped characters.
func tokenizeCurl(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '"' {
				inDouble = false
			} else if c == '\\' && i+1 < len(cmd) {
				i++
				cur.WriteByte(cmd[i])
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '\\' && i+1 < len(cmd):
			// line-continuation or escape outside quotes
			next := cmd[i+1]
			if next == '\n' || next == '\r' {
				i++ // skip the newline
			} else {
				i++
				cur.WriteByte(next)
			}
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// HTTPCurlConvert parses a curl command and generates a code snippet.
func HTTPCurlConvert(ctx context.Context, input HTTPCurlConvertInput) string {
	if strings.TrimSpace(input.Curl) == "" {
		return errResult("curl is required")
	}
	target := strings.ToLower(strings.TrimSpace(input.Target))
	switch target {
	case "go", "typescript", "python":
	default:
		return errResult("target must be one of: go, typescript, python")
	}

	parsed := parseCurl(input.Curl)
	if parsed.url == "" {
		return errResult("could not parse URL from curl command")
	}

	var snippet string
	switch target {
	case "go":
		snippet = buildGoSnippet(parsed)
	case "typescript":
		snippet = buildTypeScriptSnippet(parsed)
	case "python":
		snippet = buildPythonSnippet(parsed)
	}

	return resultJSON(HTTPCurlConvertOutput{
		Snippet: snippet,
		Target:  target,
	})
}

func buildGoSnippet(p parsedCurl) string {
	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n")
	if p.body != "" {
		sb.WriteString("\t\"strings\"\n")
	}
	sb.WriteString(")\n\n")
	sb.WriteString("func main() {\n")

	if p.body != "" {
		sb.WriteString(fmt.Sprintf("\tbody := strings.NewReader(%q)\n", p.body))
		sb.WriteString(fmt.Sprintf("\treq, _ := http.NewRequest(%q, %q, body)\n", p.method, p.url))
	} else {
		sb.WriteString(fmt.Sprintf("\treq, _ := http.NewRequest(%q, %q, nil)\n", p.method, p.url))
	}

	for k, v := range p.headers {
		sb.WriteString(fmt.Sprintf("\treq.Header.Set(%q, %q)\n", k, v))
	}

	sb.WriteString("\n\tclient := &http.Client{}\n")
	sb.WriteString("\tresp, err := client.Do(req)\n")
	sb.WriteString("\tif err != nil {\n\t\tfmt.Println(\"Error:\", err)\n\t\treturn\n\t}\n")
	sb.WriteString("\tdefer resp.Body.Close()\n")
	sb.WriteString("\tbody2, _ := io.ReadAll(resp.Body)\n")
	sb.WriteString("\tfmt.Println(resp.Status)\n")
	sb.WriteString("\tfmt.Println(string(body2))\n")
	sb.WriteString("}\n")
	return sb.String()
}

func buildTypeScriptSnippet(p parsedCurl) string {
	var sb strings.Builder
	sb.WriteString("const response = await fetch(")
	sb.WriteString(fmt.Sprintf("%q", p.url))
	sb.WriteString(", {\n")
	sb.WriteString(fmt.Sprintf("  method: %q,\n", p.method))

	if len(p.headers) > 0 {
		sb.WriteString("  headers: {\n")
		for k, v := range p.headers {
			sb.WriteString(fmt.Sprintf("    %q: %q,\n", k, v))
		}
		sb.WriteString("  },\n")
	}

	if p.body != "" {
		sb.WriteString(fmt.Sprintf("  body: %q,\n", p.body))
	}

	sb.WriteString("});\n\n")
	sb.WriteString("const data = await response.text();\n")
	sb.WriteString("console.log(response.status, data);\n")
	return sb.String()
}

func buildPythonSnippet(p parsedCurl) string {
	var sb strings.Builder
	sb.WriteString("import requests\n\n")

	if len(p.headers) > 0 {
		sb.WriteString("headers = {\n")
		for k, v := range p.headers {
			sb.WriteString(fmt.Sprintf("    %q: %q,\n", k, v))
		}
		sb.WriteString("}\n\n")
	} else {
		sb.WriteString("headers = {}\n\n")
	}

	methodLower := strings.ToLower(p.method)
	if p.body != "" {
		sb.WriteString(fmt.Sprintf("response = requests.%s(\n    %q,\n    headers=headers,\n    data=%q,\n)\n",
			methodLower, p.url, p.body))
	} else {
		sb.WriteString(fmt.Sprintf("response = requests.%s(\n    %q,\n    headers=headers,\n)\n",
			methodLower, p.url))
	}

	sb.WriteString("\nprint(response.status_code)\n")
	sb.WriteString("print(response.text)\n")
	return sb.String()
}

// ─── http_webhook_replay ──────────────────────────────────────────────────

// HTTPWebhookReplayInput is the input for the http_webhook_replay tool.
type HTTPWebhookReplayInput struct {
	URL            string            `json:"url"`
	Method         string            `json:"method"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body"`
	TimeoutSeconds int               `json:"timeout_seconds"`
}

// HTTPWebhookReplayOutput is the output for the http_webhook_replay tool.
type HTTPWebhookReplayOutput struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// HTTPWebhookReplay replays a saved webhook payload to a target URL.
func HTTPWebhookReplay(ctx context.Context, input HTTPWebhookReplayInput) string {
	if strings.TrimSpace(input.URL) == "" {
		return errResult("url is required")
	}

	method := strings.ToUpper(input.Method)
	if method == "" {
		method = "POST"
	}

	timeoutSecs := input.TimeoutSeconds
	if timeoutSecs <= 0 {
		timeoutSecs = 30
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSecs) * time.Second,
	}

	var bodyReader io.Reader
	if input.Body != "" {
		bodyReader = strings.NewReader(input.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, input.URL, bodyReader)
	if err != nil {
		return errResult("failed to create request: " + err.Error())
	}

	for k, v := range input.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return errResult("request failed: " + err.Error())
	}
	defer resp.Body.Close()

	limitedReader := io.LimitReader(resp.Body, maxBodyBytes)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return errResult("failed to read response body: " + err.Error())
	}

	headers := make(map[string]string, len(resp.Header))
	for k, vals := range resp.Header {
		headers[k] = strings.Join(vals, ", ")
	}

	return resultJSON(HTTPWebhookReplayOutput{
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    string(bodyBytes),
	})
}

// ─── http_signed_url ──────────────────────────────────────────────────────

// HTTPSignedURLInput is the input for the http_signed_url tool.
type HTTPSignedURLInput struct {
	URL           string `json:"url"`
	Secret        string `json:"secret"`
	ExpirySeconds int    `json:"expiry_seconds"`
	Method        string `json:"method"`
}

// HTTPSignedURLOutput is the output for query method.
type HTTPSignedURLOutput struct {
	SignedURL string `json:"signed_url,omitempty"`
	Signature string `json:"signature,omitempty"`
	ExpiresAt string `json:"expires_at"`
}

// HTTPSignedURL generates a signed URL or signature header value using HMAC-SHA256.
func HTTPSignedURL(ctx context.Context, input HTTPSignedURLInput) string {
	if strings.TrimSpace(input.URL) == "" {
		return errResult("url is required")
	}
	if strings.TrimSpace(input.Secret) == "" {
		return errResult("secret is required")
	}

	method := strings.ToLower(input.Method)
	if method == "" {
		method = "query"
	}
	if method != "query" && method != "header" {
		return errResult("method must be query or header")
	}

	expirySecs := input.ExpirySeconds
	if expirySecs <= 0 {
		expirySecs = 3600
	}

	expiresAt := time.Now().Add(time.Duration(expirySecs) * time.Second)
	expiresUnix := expiresAt.Unix()

	parsedURL, err := url.Parse(input.URL)
	if err != nil {
		return errResult("invalid url: " + err.Error())
	}

	// Build the message to sign: path + "?expires=" + timestamp
	pathAndExpiry := parsedURL.Path + fmt.Sprintf("?expires=%d", expiresUnix)

	mac := hmac.New(sha256.New, []byte(input.Secret))
	mac.Write([]byte(pathAndExpiry))
	sig := hex.EncodeToString(mac.Sum(nil))

	expiresAtISO := expiresAt.UTC().Format(time.RFC3339)

	if method == "query" {
		q := parsedURL.Query()
		q.Set("expires", fmt.Sprintf("%d", expiresUnix))
		q.Set("signature", sig)
		parsedURL.RawQuery = q.Encode()

		return resultJSON(HTTPSignedURLOutput{
			SignedURL: parsedURL.String(),
			ExpiresAt: expiresAtISO,
		})
	}

	// header method: return the signature value (caller adds it as a header)
	return resultJSON(HTTPSignedURLOutput{
		Signature: sig,
		ExpiresAt: expiresAtISO,
	})
}

// ─── http_url_parse ───────────────────────────────────────────────────────

// HTTPURLParseInput is the input for the http_url_parse tool.
type HTTPURLParseInput struct {
	URL        string                 `json:"url"`
	Action     string                 `json:"action"`
	Components map[string]interface{} `json:"components"`
}

// HTTPURLParseOutput is the output for the parse action.
type HTTPURLParseOutput struct {
	Scheme   string            `json:"scheme"`
	Host     string            `json:"host"`
	Port     string            `json:"port"`
	Path     string            `json:"path"`
	Query    map[string]string `json:"query"`
	Fragment string            `json:"fragment"`
	RawQuery string            `json:"raw_query"`
}

// HTTPURLBuildOutput is the output for the build action.
type HTTPURLBuildOutput struct {
	URL string `json:"url"`
}

// HTTPURLParse parses or builds a URL.
func HTTPURLParse(ctx context.Context, input HTTPURLParseInput) string {
	action := strings.ToLower(input.Action)
	if action == "" {
		action = "parse"
	}

	switch action {
	case "parse":
		return httpURLParseAction(input.URL)
	case "build":
		return httpURLBuildAction(input.Components)
	default:
		return errResult("action must be parse or build")
	}
}

func httpURLParseAction(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return errResult("url is required for parse action")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errResult("invalid url: " + err.Error())
	}

	port := parsed.Port()
	host := parsed.Hostname()

	queryMap := make(map[string]string)
	for k, vals := range parsed.Query() {
		if len(vals) > 0 {
			queryMap[k] = vals[0]
		}
	}

	return resultJSON(HTTPURLParseOutput{
		Scheme:   parsed.Scheme,
		Host:     host,
		Port:     port,
		Path:     parsed.Path,
		Query:    queryMap,
		Fragment: parsed.Fragment,
		RawQuery: parsed.RawQuery,
	})
}

func httpURLBuildAction(components map[string]interface{}) string {
	if components == nil {
		return errResult("components are required for build action")
	}

	strField := func(key string) string {
		if v, ok := components[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	u := &url.URL{
		Scheme:   strField("scheme"),
		Host:     strField("host"),
		Path:     strField("path"),
		Fragment: strField("fragment"),
	}

	// Merge query parameters
	if q, ok := components["query"]; ok {
		switch qv := q.(type) {
		case map[string]interface{}:
			params := url.Values{}
			for k, v := range qv {
				if s, ok := v.(string); ok {
					params.Set(k, s)
				}
			}
			u.RawQuery = params.Encode()
		case map[string]string:
			params := url.Values{}
			for k, v := range qv {
				params.Set(k, v)
			}
			u.RawQuery = params.Encode()
		}
	}

	return resultJSON(HTTPURLBuildOutput{URL: u.String()})
}
