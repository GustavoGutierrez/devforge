# Data Formatting Tools (`datafmt`)

MCP tools for parsing, converting, querying, validating, and diffing structured data (JSON, YAML, CSV).

---

## Tools

### `data_json_format`

Pretty-print or re-indent a JSON string.

**Input**

| Parameter | Type   | Required | Default | Description                                      |
|-----------|--------|----------|---------|--------------------------------------------------|
| `json`    | string | ✅        | —       | Raw JSON string to format                        |
| `indent`  | string |          | `"  "`  | Indent string (e.g. `"  "`, `"\t"`, `"    "`)   |

**Output (success)**

```json
{ "result": "<pretty-printed json>" }
```

**Output (syntax error)**

```json
{ "error": "<message>", "line": 3, "column": 12 }
```

**Example**

```json
// Input
{ "json": "{\"b\":2,\"a\":1}", "indent": "  " }

// Output
{ "result": "{\n  \"b\": 2,\n  \"a\": 1\n}" }
```

---

### `data_yaml_convert`

Convert data between JSON and YAML formats.

**Input**

| Parameter | Type   | Required | Description                         |
|-----------|--------|----------|-------------------------------------|
| `input`   | string | ✅        | Input string to convert             |
| `from`    | string | ✅        | Source format: `json` \| `yaml`     |
| `to`      | string | ✅        | Target format: `json` \| `yaml`     |

**Output**

```json
{ "result": "<converted string>" }
```

**Example — JSON → YAML**

```json
// Input
{ "input": "{\"name\":\"alice\",\"age\":30}", "from": "json", "to": "yaml" }

// Output
{ "result": "age: 30\nname: alice" }
```

**Example — YAML → JSON**

```json
// Input
{ "input": "name: alice\nage: 30\n", "from": "yaml", "to": "json" }

// Output
{ "result": "{\n  \"age\": 30,\n  \"name\": \"alice\"\n}" }
```

---

### `data_csv_convert`

Convert between CSV and JSON formats.

- **CSV → JSON**: produces an array of objects (keys from header row, or `"0"`, `"1"`, … if `has_header` is `false`).
- **JSON → CSV**: takes an array of objects and outputs a CSV with a header row.

**Input**

| Parameter    | Type    | Required | Default | Description                                  |
|--------------|---------|----------|---------|----------------------------------------------|
| `input`      | string  | ✅        | —       | Input string to convert                      |
| `from`       | string  | ✅        | —       | Source format: `csv` \| `json`               |
| `to`         | string  | ✅        | —       | Target format: `csv` \| `json`               |
| `separator`  | string  |          | `,`     | Single-character field separator             |
| `has_header` | boolean |          | `true`  | Whether the CSV has a header row             |

**Output**

```json
{ "result": "<converted string>" }
```

**Example — CSV → JSON**

```
// Input CSV
name,age
alice,30
bob,25

// Output JSON
[
  { "age": "30", "name": "alice" },
  { "age": "25", "name": "bob" }
]
```

**Example — JSON → CSV**

```json
// Input JSON
[{"name":"alice","age":"30"},{"name":"bob","age":"25"}]

// Output CSV
age,name
30,alice
25,bob
```

> **Note:** All CSV cell values are strings. Numeric conversion is not performed.

---

### `data_jsonpath`

Evaluate a JSONPath expression against a JSON document.

**Supported syntax**

| Token   | Description                             |
|---------|-----------------------------------------|
| `$`     | Root element (required at start)        |
| `.field` | Access object property `field`         |
| `[N]`   | Access array element at index `N`       |
| `.*`    | All values of an object                 |
| `[*]`   | All elements of an array                |

**Input**

| Parameter | Type   | Required | Description                                          |
|-----------|--------|----------|------------------------------------------------------|
| `json`    | string | ✅        | JSON document to query                               |
| `path`    | string | ✅        | JSONPath expression (e.g. `$.store.book[0].title`)  |

**Output**

```json
{ "result": <extracted value> }
```

If a single match is found, `result` is the value directly. If multiple values match (wildcard), `result` is an array.

**Examples**

```json
// $.store.book[0].title
{ "result": "Go Programming" }

// $.items[*]  (wildcard)
{ "result": [1, 2, 3] }

// $.*  (all object values)
{ "result": ["bookstore", [...]] }
```

---

### `data_schema_validate`

Validate a JSON document against a JSON Schema (basic subset — no external libraries).

**Supported keywords**

| Category | Keywords                                                                 |
|----------|--------------------------------------------------------------------------|
| Any type | `type`, `enum`                                                           |
| String   | `minLength`, `maxLength`, `pattern`                                      |
| Number   | `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`             |
| Object   | `required`, `properties`, `additionalProperties`                         |
| Array    | `items`, `minItems`, `maxItems`                                          |

**Input**

| Parameter | Type   | Required | Description             |
|-----------|--------|----------|-------------------------|
| `json`    | string | ✅        | JSON document to validate |
| `schema`  | string | ✅        | JSON Schema document    |

**Output (valid)**

```json
{ "valid": true }
```

**Output (invalid)**

```json
{
  "valid": false,
  "errors": [
    "$.age: expected type \"number\", got string",
    "$.name: missing required property \"name\""
  ]
}
```

**Example**

```json
// Schema
{
  "type": "object",
  "required": ["name", "age"],
  "properties": {
    "name": { "type": "string", "minLength": 2 },
    "age":  { "type": "number", "minimum": 0, "maximum": 150 }
  }
}

// Valid document → { "valid": true }
{ "name": "Alice", "age": 30 }

// Invalid document → { "valid": false, "errors": ["..."] }
{ "name": "A", "age": -1 }
```

---

### `data_diff`

Structural diff between two JSON or YAML objects at the top-level key level.

**Input**

| Parameter | Type   | Required | Default | Description                                 |
|-----------|--------|----------|---------|---------------------------------------------|
| `a`       | string | ✅        | —       | First document (JSON or YAML)               |
| `b`       | string | ✅        | —       | Second document (JSON or YAML)              |
| `format`  | string |          | `json`  | Input format: `json` \| `yaml`              |

**Output**

```json
{
  "added":   ["country"],
  "removed": ["city"],
  "changed": [
    { "key": "age", "from": 30, "to": 31 }
  ]
}
```

- `added`: keys present in `b` but not in `a`
- `removed`: keys present in `a` but not in `b`
- `changed`: keys present in both with different values; each entry has `key`, `from`, `to`

**Example**

```json
// a
{ "name": "alice", "age": 30, "city": "NY" }

// b
{ "name": "alice", "age": 31, "country": "US" }

// Output
{
  "added":   ["country"],
  "removed": ["city"],
  "changed": [{ "key": "age", "from": 30, "to": 31 }]
}
```

> **Note:** The diff is shallow — it compares top-level keys only. Nested changes are reported as a single `changed` entry with the full old/new sub-tree.

---

## Error Responses

All tools return a JSON error object on failure:

```json
{ "error": "<human-readable message>" }
```

`data_json_format` additionally returns `line` and `column` fields when the input has a syntax error:

```json
{ "error": "unexpected end of JSON input", "line": 2, "column": 15 }
```

---

## Implementation Notes

- No external libraries beyond `gopkg.in/yaml.v3` and the Go standard library.
- JSONPath evaluator is minimal: `$`, `.field`, `[N]`, `.*`, `[*]` — no filter expressions, recursive descent, or script expressions.
- JSON Schema validation covers the most common keywords; `$ref`, `allOf`, `anyOf`, `oneOf`, `not` are not supported.
- CSV→JSON values are always strings (no type inference).
- `data_diff` compares top-level keys only using JSON-serialized equality for value comparison.
