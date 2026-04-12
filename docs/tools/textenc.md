# textenc — Text & Encoding Tools

Package `internal/tools/textenc` — MCP tool group for stateless text manipulation and encoding operations.

---

## Tools

### `text_escape`

Escape or unescape a string for a specific syntax target.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Input string to escape or unescape |
| `target` | string | no | `json` | Escaping target: `json` \| `js` \| `html` \| `sql` |
| `operation` | string | no | `escape` | Operation: `escape` \| `unescape` |

**Return schema**

```json
{ "result": "<escaped or unescaped string>" }
```

**Error**

```json
{ "error": "<description>" }
```

**Escape rules by target**

| Target | Escape | Unescape |
|--------|--------|---------|
| `json` | `encoding/json` marshal without outer quotes | `json.Unmarshal` of a quoted string |
| `html` | `html.EscapeString` (`<`, `>`, `&`, `"`, `'`) | `html.UnescapeString` |
| `js` | backslash, double-quote, single-quote, `\n`, `\r`, `\t` | reverse substitutions |
| `sql` | double single-quotes (`'` → `''`) | halve doubled single-quotes (`''` → `'`) |

**Example**

```json
{ "text": "hello\nworld", "target": "json", "operation": "escape" }
→ { "result": "hello\\nworld" }
```

---

### `text_slug`

Convert arbitrary text to a URL-safe slug.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Input text to slugify |
| `separator` | string | no | `-` | Word separator character |
| `lower` | bool | no | `true` | Convert to lowercase |

**Return schema**

```json
{ "slug": "<url-safe-slug>" }
```

**Algorithm**

1. NFC-normalize the input to pre-compose combining characters.
2. Map common non-ASCII Latin characters to ASCII equivalents (`é` → `e`, `ü` → `u`, `ß` → `ss`, etc.).
3. Optionally lowercase.
4. Replace any run of non-alphanumeric characters with the separator.
5. Trim leading/trailing separators.

**Example**

```json
{ "text": "Héllo Wörld!", "separator": "-", "lower": true }
→ { "slug": "hello-world" }
```

---

### `text_uuid`

Generate a unique identifier.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `kind` | string | no | `uuid4` | Kind: `uuid4` \| `ulid` \| `nanoid` \| `token` |
| `length` | number | no | `21` | Length for `nanoid` and `token` |
| `count` | number | no | `1` | Number of identifiers to generate (max `1000`) |

**Return schema**

When `count = 1`:

```json
{ "value": "<generated identifier>" }
```

When `count > 1`:

```json
{
  "values": ["<id1>", "<id2>", "..."],
  "count": 3,
  "kind": "uuid4"
}
```

**Kind details**

| Kind | Description | Output example |
|------|-------------|----------------|
| `uuid4` | RFC 4122 UUID v4 via `github.com/google/uuid` | `f47ac10b-58cc-4372-a567-0e02b2c3d479` |
| `ulid` | Time-ordered ULID (Crockford Base32, 26 chars) | `01JZ9WX2RZ1N1M6N6SK2D1XQ0E` |
| `nanoid` | Random URL-safe string from alphabet `A-Za-z0-9_-` | `V1StGXR8_Z5jdHi6B-myT` |
| `token` | Hex-encoded cryptographically random bytes; output length = 2 × `length` | `a3f4b9c1...` |

All kinds use `crypto/rand` for entropy.

**Example**

```json
{ "kind": "nanoid", "length": 16 }
→ { "value": "V1StGXR8_Z5jdHi6" }
```

---

### `text_base64`

Encode or decode a string using Base64.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Input string to encode or decode |
| `variant` | string | no | `standard` | Base64 variant: `standard` \| `urlsafe` |
| `operation` | string | no | `encode` | Operation: `encode` \| `decode` |

**Return schema**

```json
{ "result": "<base64 or decoded string>" }
```

**Variants**

| Variant | Alphabet | Padding |
|---------|----------|---------|
| `standard` | `A-Za-z0-9+/` | `=` |
| `urlsafe` | `A-Za-z0-9-_` | `=` |

Decode accepts both padded and unpadded inputs.

**Example**

```json
{ "text": "hello world", "variant": "standard", "operation": "encode" }
→ { "result": "aGVsbG8gd29ybGQ=" }
```

---

### `text_url_encode`

Percent-encode or decode a URL component.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Input string to encode or decode |
| `operation` | string | no | `encode` | Operation: `encode` \| `decode` |
| `mode` | string | no | `query` | Encoding mode: `query` \| `path` |

**Return schema**

```json
{ "result": "<encoded or decoded string>" }
```

**Mode differences**

| Mode | Space encoding | Slashes |
|------|---------------|---------|
| `query` | `+` | `%2F` |
| `path` | `%20` | `%2F` |

Uses `url.QueryEscape`/`url.QueryUnescape` for `query` mode and `url.PathEscape`/`url.PathUnescape` for `path` mode.

**Example**

```json
{ "text": "hello world & more", "operation": "encode", "mode": "query" }
→ { "result": "hello+world+%26+more" }
```

---

### `text_normalize`

Apply one or more normalization operations to text.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Input text to normalize |
| `operations` | string[] | yes | — | Ordered list of operations to apply |

**Valid operations**

| Operation | Effect |
|-----------|--------|
| `trim_whitespace` | Remove leading and trailing whitespace |
| `normalize_newlines` | Normalize `\r\n` and bare `\r` to `\n` |
| `strip_bom` | Remove UTF-8 BOM (`\xef\xbb\xbf`) if present |
| `nfc` | Unicode NFC normalization (canonical decomposition then canonical composition) |
| `nfd` | Unicode NFD normalization (canonical decomposition) |
| `nfkc` | Unicode NFKC normalization (compatibility decomposition then canonical composition) |
| `nfkd` | Unicode NFKD normalization (compatibility decomposition) |

Operations are applied in the order they appear in the array.

**Return schema**

```json
{ "result": "<normalized text>" }
```

**Example**

```json
{ "text": "  \r\nhello\r\n  ", "operations": ["normalize_newlines", "trim_whitespace"] }
→ { "result": "hello" }
```

---

### `text_case`

Convert text between naming conventions.

**Parameters**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `text` | string | yes | — | Input text to convert |
| `target_case` | string | yes | — | Target naming convention |

**Valid target_case values**

| Value | Example output |
|-------|---------------|
| `camel` | `helloWorldFoo` |
| `pascal` | `HelloWorldFoo` |
| `snake` | `hello_world_foo` |
| `kebab` | `hello-world-foo` |
| `screaming_snake` | `HELLO_WORLD_FOO` |

**Tokenization rules** — the following are all treated as word boundaries:
- Spaces
- Underscores (`_`)
- Hyphens (`-`)
- Lowercase-to-uppercase transitions (camelCase and PascalCase boundaries)

**Return schema**

```json
{ "result": "<converted text>" }
```

**Example**

```json
{ "text": "helloWorldFoo", "target_case": "snake" }
→ { "result": "hello_world_foo" }
```

---

## Implementation Notes

- All functions are pure and stateless — safe for concurrent use.
- No external network calls.
- No database reads or writes.
- `crypto/rand` is used for all random generation (`text_uuid`).
- Unicode normalization uses `golang.org/x/text/unicode/norm`.
- UUID v4 generation uses `github.com/google/uuid`.
