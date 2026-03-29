# httptools — HTTP & Networking Tools

Tools for making HTTP requests, converting curl commands to code snippets, replaying webhooks, generating signed URLs, and parsing/building URLs.

---

## `http_request`

Perform an HTTP request and return the status code, response headers, response body, and request duration.

### Input

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | ✅ | — | Target URL |
| `method` | string | — | `GET` | HTTP method |
| `headers` | object | — | `{}` | Request headers as key/value string pairs |
| `body` | string | — | `""` | Request body |
| `timeout_seconds` | int | — | `30` | Timeout in seconds |
| `follow_redirects` | bool | — | `true` | Whether to follow HTTP redirects |

### Output

```json
{
  "status": 200,
  "headers": {
    "Content-Type": "application/json",
    "X-Request-Id": "abc123"
  },
  "body": "{\"hello\":\"world\"}",
  "duration_ms": 142
}
```

### Notes

- Response body is capped at **1 MB**.
- On network or timeout errors, returns `{"error": "..."}`.

---

## `http_curl_convert`

Parse a `curl` command and generate an idiomatic code snippet in Go, TypeScript, or Python.

### Input

| Parameter | Type | Required | Description |
|---|---|---|---|
| `curl` | string | ✅ | The curl command string |
| `target` | string | ✅ | Target language: `go` \| `typescript` \| `python` |

### Output

```json
{
  "snippet": "const response = await fetch(\"https://api.example.com\", {\n  method: \"POST\",\n  ...\n});\n",
  "target": "typescript"
}
```

### Supported curl flags

| Flag | Meaning |
|---|---|
| `-X` / `--request` | HTTP method |
| `-H` / `--header` | Request header |
| `-d` / `--data` / `--data-raw` / `--data-binary` | Request body (also implies POST) |
| `--url` | Explicit URL |
| Positional arg | URL (if no `--url`) |

Full curl compatibility is not attempted — only the flags listed above are parsed.

---

## `http_webhook_replay`

Replay a saved webhook payload to a new or updated URL.

### Input

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | ✅ | — | Target URL |
| `method` | string | — | `POST` | HTTP method |
| `headers` | object | — | `{}` | Request headers as key/value string pairs |
| `body` | string | — | `""` | Webhook payload body |
| `timeout_seconds` | int | — | `30` | Timeout in seconds |

### Output

```json
{
  "status": 200,
  "headers": {
    "Content-Type": "application/json"
  },
  "body": "OK"
}
```

---

## `http_signed_url`

Generate a signed URL or HMAC-SHA256 signature for time-limited, secure access to a resource.

### Input

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | ✅ | — | URL to sign |
| `secret` | string | ✅ | — | HMAC-SHA256 secret key |
| `expiry_seconds` | int | — | `3600` | Seconds until the signature expires |
| `method` | string | — | `query` | Delivery method: `query` \| `header` |

### Output (query method)

```json
{
  "signed_url": "https://cdn.example.com/file.jpg?expires=1748000000&signature=abc123...",
  "expires_at": "2025-05-23T12:00:00Z"
}
```

### Output (header method)

```json
{
  "signature": "abc123...",
  "expires_at": "2025-05-23T12:00:00Z"
}
```

### Signing algorithm

```
message = URL_PATH + "?expires=" + UNIX_TIMESTAMP
signature = HMAC-SHA256(message, secret)  →  hex-encoded
```

For the `query` method the `expires` and `signature` parameters are appended to the URL's query string. For the `header` method, the caller is responsible for attaching the `signature` value as a request header.

---

## `http_url_parse`

Parse a URL into its components or build a URL from components.

### Input

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | — | — | URL to parse (required for `parse` action) |
| `action` | string | — | `parse` | Action: `parse` \| `build` |
| `components` | object | — | — | URL parts for `build` action |

#### `components` object (for `build`)

| Field | Type | Description |
|---|---|---|
| `scheme` | string | e.g. `https` |
| `host` | string | e.g. `api.example.com` or `api.example.com:8443` |
| `path` | string | e.g. `/v1/users` |
| `query` | object | Key/value string pairs for query parameters |
| `fragment` | string | URL fragment (without `#`) |

### Output (parse)

```json
{
  "scheme": "https",
  "host": "api.example.com",
  "port": "8443",
  "path": "/v1/users",
  "query": {
    "page": "2",
    "limit": "20"
  },
  "fragment": "section",
  "raw_query": "page=2&limit=20"
}
```

### Output (build)

```json
{
  "url": "https://api.example.com/v1/items?page=1&sort=desc#top"
}
```
