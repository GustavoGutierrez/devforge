package httptools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dev-forge-mcp/internal/tools/httptools"
)

// ─── Helpers ──────────────────────────────────────────────────────────────

func parseJSON(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, s)
	}
	return m
}

func assertNoError(t *testing.T, result string) {
	t.Helper()
	m := parseJSON(t, result)
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error in result: %s", result)
	}
}

func assertError(t *testing.T, result, wantContains string) {
	t.Helper()
	m := parseJSON(t, result)
	errVal, ok := m["error"]
	if !ok {
		t.Fatalf("expected error key in result: %s", result)
	}
	if wantContains != "" && !strings.Contains(errVal.(string), wantContains) {
		t.Fatalf("expected error containing %q, got %q", wantContains, errVal)
	}
}

// ─── HTTPRequest ──────────────────────────────────────────────────────────

func TestHTTPRequest_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test-value")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	result := httptools.HTTPRequest(context.Background(), httptools.HTTPRequestInput{
		URL:             srv.URL,
		Method:          "GET",
		TimeoutSeconds:  5,
		FollowRedirects: true,
	})

	assertNoError(t, result)

	m := parseJSON(t, result)
	if status, ok := m["status"].(float64); !ok || int(status) != 200 {
		t.Errorf("expected status 200, got %v", m["status"])
	}
	if body, ok := m["body"].(string); !ok || !strings.Contains(body, "hello") {
		t.Errorf("expected body to contain 'hello', got %q", m["body"])
	}
	if headers, ok := m["headers"].(map[string]interface{}); !ok {
		t.Error("expected headers map in response")
	} else if _, ok := headers["X-Custom"]; !ok {
		t.Error("expected X-Custom header in response")
	}
	if _, ok := m["duration_ms"]; !ok {
		t.Error("expected duration_ms in response")
	}
}

func TestHTTPRequest_ErrorPath_EmptyURL(t *testing.T) {
	result := httptools.HTTPRequest(context.Background(), httptools.HTTPRequestInput{
		URL: "",
	})
	assertError(t, result, "url is required")
}

func TestHTTPRequest_WithHeaders(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	result := httptools.HTTPRequest(context.Background(), httptools.HTTPRequestInput{
		URL:    srv.URL,
		Method: "GET",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
		TimeoutSeconds: 5,
	})

	assertNoError(t, result)
	if gotAuth != "Bearer token123" {
		t.Errorf("expected Authorization header to be sent, got %q", gotAuth)
	}
}

func TestHTTPRequest_PostWithBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 1024)
		n, _ := r.Body.Read(b)
		gotBody = string(b[:n])
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	result := httptools.HTTPRequest(context.Background(), httptools.HTTPRequestInput{
		URL:    srv.URL,
		Method: "POST",
		Body:   `{"key":"value"}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		TimeoutSeconds: 5,
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	if status := int(m["status"].(float64)); status != 201 {
		t.Errorf("expected status 201, got %d", status)
	}
	if !strings.Contains(gotBody, "key") {
		t.Errorf("expected body to be sent, got %q", gotBody)
	}
}

func TestHTTPRequest_NoFollowRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer srv.Close()

	result := httptools.HTTPRequest(context.Background(), httptools.HTTPRequestInput{
		URL:             srv.URL,
		Method:          "GET",
		TimeoutSeconds:  5,
		FollowRedirects: false,
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	if status := int(m["status"].(float64)); status != 302 {
		t.Errorf("expected 302 (no follow), got %d", status)
	}
}

// ─── HTTPCurlConvert ──────────────────────────────────────────────────────

func TestHTTPCurlConvert_HappyPath_Go(t *testing.T) {
	result := httptools.HTTPCurlConvert(context.Background(), httptools.HTTPCurlConvertInput{
		Curl:   `curl -X POST https://api.example.com/data -H "Content-Type: application/json" -d '{"key":"value"}'`,
		Target: "go",
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	snippet, _ := m["snippet"].(string)
	if !strings.Contains(snippet, "http.NewRequest") {
		t.Errorf("expected Go snippet to contain http.NewRequest, got:\n%s", snippet)
	}
	if !strings.Contains(snippet, "api.example.com") {
		t.Errorf("expected snippet to contain URL, got:\n%s", snippet)
	}
	if m["target"] != "go" {
		t.Errorf("expected target=go, got %v", m["target"])
	}
}

func TestHTTPCurlConvert_HappyPath_TypeScript(t *testing.T) {
	result := httptools.HTTPCurlConvert(context.Background(), httptools.HTTPCurlConvertInput{
		Curl:   `curl https://api.example.com/users`,
		Target: "typescript",
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	snippet, _ := m["snippet"].(string)
	if !strings.Contains(snippet, "fetch") {
		t.Errorf("expected TypeScript snippet to use fetch, got:\n%s", snippet)
	}
}

func TestHTTPCurlConvert_HappyPath_Python(t *testing.T) {
	result := httptools.HTTPCurlConvert(context.Background(), httptools.HTTPCurlConvertInput{
		Curl:   `curl -X GET https://api.example.com/items -H "Accept: application/json"`,
		Target: "python",
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	snippet, _ := m["snippet"].(string)
	if !strings.Contains(snippet, "requests") {
		t.Errorf("expected Python snippet to use requests, got:\n%s", snippet)
	}
}

func TestHTTPCurlConvert_ErrorPath_EmptyCurl(t *testing.T) {
	result := httptools.HTTPCurlConvert(context.Background(), httptools.HTTPCurlConvertInput{
		Curl:   "",
		Target: "go",
	})
	assertError(t, result, "curl is required")
}

func TestHTTPCurlConvert_ErrorPath_InvalidTarget(t *testing.T) {
	result := httptools.HTTPCurlConvert(context.Background(), httptools.HTTPCurlConvertInput{
		Curl:   "curl https://example.com",
		Target: "ruby",
	})
	assertError(t, result, "target must be one of")
}

// ─── HTTPWebhookReplay ────────────────────────────────────────────────────

func TestHTTPWebhookReplay_HappyPath(t *testing.T) {
	var gotMethod string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b := make([]byte, 1024)
		n, _ := r.Body.Read(b)
		gotBody = string(b[:n])
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("received"))
	}))
	defer srv.Close()

	result := httptools.HTTPWebhookReplay(context.Background(), httptools.HTTPWebhookReplayInput{
		URL:            srv.URL,
		Method:         "POST",
		Body:           `{"event":"user.created","id":42}`,
		TimeoutSeconds: 5,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	if status := int(m["status"].(float64)); status != 200 {
		t.Errorf("expected status 200, got %d", status)
	}
	if body, _ := m["body"].(string); body != "received" {
		t.Errorf("expected body 'received', got %q", body)
	}
	if gotMethod != "POST" {
		t.Errorf("expected POST method sent, got %q", gotMethod)
	}
	if !strings.Contains(gotBody, "user.created") {
		t.Errorf("expected body payload to be sent, got %q", gotBody)
	}
}

func TestHTTPWebhookReplay_ErrorPath_EmptyURL(t *testing.T) {
	result := httptools.HTTPWebhookReplay(context.Background(), httptools.HTTPWebhookReplayInput{
		URL: "",
	})
	assertError(t, result, "url is required")
}

// ─── HTTPSignedURL ────────────────────────────────────────────────────────

func TestHTTPSignedURL_HappyPath_Query(t *testing.T) {
	result := httptools.HTTPSignedURL(context.Background(), httptools.HTTPSignedURLInput{
		URL:           "https://cdn.example.com/private/file.jpg",
		Secret:        "my-secret-key",
		ExpirySeconds: 3600,
		Method:        "query",
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	signedURL, _ := m["signed_url"].(string)
	if !strings.Contains(signedURL, "signature=") {
		t.Errorf("expected signed_url to contain signature, got %q", signedURL)
	}
	if !strings.Contains(signedURL, "expires=") {
		t.Errorf("expected signed_url to contain expires, got %q", signedURL)
	}
	if _, ok := m["expires_at"]; !ok {
		t.Error("expected expires_at in response")
	}
}

func TestHTTPSignedURL_HappyPath_Header(t *testing.T) {
	result := httptools.HTTPSignedURL(context.Background(), httptools.HTTPSignedURLInput{
		URL:           "https://cdn.example.com/private/file.jpg",
		Secret:        "my-secret-key",
		ExpirySeconds: 900,
		Method:        "header",
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	sig, _ := m["signature"].(string)
	if sig == "" {
		t.Error("expected non-empty signature for header method")
	}
	if _, ok := m["signed_url"]; ok {
		t.Error("header method should not return signed_url")
	}
}

func TestHTTPSignedURL_ErrorPath_EmptyURL(t *testing.T) {
	result := httptools.HTTPSignedURL(context.Background(), httptools.HTTPSignedURLInput{
		URL:    "",
		Secret: "secret",
	})
	assertError(t, result, "url is required")
}

func TestHTTPSignedURL_ErrorPath_EmptySecret(t *testing.T) {
	result := httptools.HTTPSignedURL(context.Background(), httptools.HTTPSignedURLInput{
		URL:    "https://example.com/file",
		Secret: "",
	})
	assertError(t, result, "secret is required")
}

func TestHTTPSignedURL_ErrorPath_InvalidMethod(t *testing.T) {
	result := httptools.HTTPSignedURL(context.Background(), httptools.HTTPSignedURLInput{
		URL:    "https://example.com/file",
		Secret: "secret",
		Method: "cookie",
	})
	assertError(t, result, "method must be")
}

// ─── HTTPURLParse ─────────────────────────────────────────────────────────

func TestHTTPURLParse_HappyPath_Parse(t *testing.T) {
	result := httptools.HTTPURLParse(context.Background(), httptools.HTTPURLParseInput{
		URL:    "https://api.example.com:8443/v1/users?page=2&limit=20#section",
		Action: "parse",
	})

	assertNoError(t, result)
	m := parseJSON(t, result)

	tests := []struct {
		field string
		want  string
	}{
		{"scheme", "https"},
		{"host", "api.example.com"},
		{"port", "8443"},
		{"path", "/v1/users"},
		{"fragment", "section"},
		{"raw_query", "page=2&limit=20"},
	}
	for _, tc := range tests {
		got, _ := m[tc.field].(string)
		if got != tc.want {
			t.Errorf("field %q: expected %q, got %q", tc.field, tc.want, got)
		}
	}

	query, ok := m["query"].(map[string]interface{})
	if !ok {
		t.Fatal("expected query to be an object")
	}
	if query["page"] != "2" {
		t.Errorf("expected query.page=2, got %v", query["page"])
	}
	if query["limit"] != "20" {
		t.Errorf("expected query.limit=20, got %v", query["limit"])
	}
}

func TestHTTPURLParse_HappyPath_Build(t *testing.T) {
	result := httptools.HTTPURLParse(context.Background(), httptools.HTTPURLParseInput{
		Action: "build",
		Components: map[string]interface{}{
			"scheme":   "https",
			"host":     "api.example.com",
			"path":     "/v1/items",
			"query":    map[string]interface{}{"sort": "desc", "page": "1"},
			"fragment": "top",
		},
	})

	assertNoError(t, result)
	m := parseJSON(t, result)
	builtURL, _ := m["url"].(string)
	if !strings.HasPrefix(builtURL, "https://api.example.com/v1/items") {
		t.Errorf("expected built URL to start with https://api.example.com/v1/items, got %q", builtURL)
	}
	if !strings.Contains(builtURL, "sort=desc") {
		t.Errorf("expected built URL to contain sort=desc, got %q", builtURL)
	}
}

func TestHTTPURLParse_ErrorPath_EmptyURL(t *testing.T) {
	result := httptools.HTTPURLParse(context.Background(), httptools.HTTPURLParseInput{
		URL:    "",
		Action: "parse",
	})
	assertError(t, result, "url is required")
}

func TestHTTPURLParse_ErrorPath_InvalidAction(t *testing.T) {
	result := httptools.HTTPURLParse(context.Background(), httptools.HTTPURLParseInput{
		URL:    "https://example.com",
		Action: "delete",
	})
	assertError(t, result, "action must be")
}

func TestHTTPURLParse_ErrorPath_BuildWithoutComponents(t *testing.T) {
	result := httptools.HTTPURLParse(context.Background(), httptools.HTTPURLParseInput{
		Action:     "build",
		Components: nil,
	})
	assertError(t, result, "components are required")
}

// ─── parseCurl internal ───────────────────────────────────────────────────

func TestHTTPCurlConvert_ParsesHeadersAndMethod(t *testing.T) {
	cases := []struct {
		name   string
		curl   string
		target string
		checks func(t *testing.T, snippet string)
	}{
		{
			name:   "go with POST and header",
			curl:   `curl -X POST https://api.test.com/endpoint -H "Authorization: Bearer tok"`,
			target: "go",
			checks: func(t *testing.T, s string) {
				if !strings.Contains(s, "POST") {
					t.Error("expected POST in snippet")
				}
				if !strings.Contains(s, "Authorization") {
					t.Error("expected Authorization header in snippet")
				}
			},
		},
		{
			name:   "python with data implies POST",
			curl:   `curl https://api.test.com/login -d '{"user":"bob"}'`,
			target: "python",
			checks: func(t *testing.T, s string) {
				if !strings.Contains(s, "post") {
					t.Error("expected post method in python snippet")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := httptools.HTTPCurlConvert(context.Background(), httptools.HTTPCurlConvertInput{
				Curl:   tc.curl,
				Target: tc.target,
			})
			assertNoError(t, result)
			m := parseJSON(t, result)
			snippet, _ := m["snippet"].(string)
			tc.checks(t, snippet)
		})
	}
}
