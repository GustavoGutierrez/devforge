# DevForge MCP — Technical Reference & Usage Guide

**Document type:** Technical Reference  
**Version:** 2.1.2  
**Date:** 2026-04-05  
**Scope:** Developer tooling integration via Model Context Protocol  
**Author:** Ing. Gustavo Gutiérrez — Bogotá, Colombia

---

## Table of Contents

1. [Abstract](#1-abstract)
2. [Overview](#2-overview)
3. [Problem Statement](#3-problem-statement)
4. [Capabilities & Tool Taxonomy](#4-capabilities--tool-taxonomy)
5. [Integration Value Analysis](#5-integration-value-analysis)
6. [Practical Examples](#6-practical-examples)
   - 6.1 [Cryptographic Hashing — `crypto_hash`](#61-cryptographic-hashing--crypto_hash)
   - 6.2 [SQL Formatting & Linting — `backend_sql_format`](#62-sql-formatting--linting--backend_sql_format)
   - 6.3 [Data Format Conversion — `data_yaml_convert`](#63-data-format-conversion--data_yaml_convert)
   - 6.4 [JSONPath Querying — `data_jsonpath`](#64-jsonpath-querying--data_jsonpath)
   - 6.5 [Temporal Sequence Generation — `time_date_range`](#65-temporal-sequence-generation--time_date_range)
7. [Role-Based Applicability Matrix](#7-role-based-applicability-matrix)
8. [Conclusions](#8-conclusions)
9. [References](#references)

---

## 1. Abstract

This document provides a technical reference for **DevForge MCP**, a Model Context Protocol server that exposes over 70 deterministic, side-effect-free developer utilities as callable tools within an AI-agent conversation context. The integration eliminates context-switching overhead by enabling cryptographic operations, data transformation, image processing, SQL formatting, and HTTP utilities to be invoked directly within the development workflow — without requiring external tooling, runtime dependencies, or manual scripting.

---

## 2. Overview

**DevForge MCP** implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) specification, which defines a standardized interface for exposing executable tools to language model agents. Each tool is declared as a typed function schema (JSON Schema) and invoked deterministically by the agent runtime.

Key characteristics:

- **Stateless execution** — each tool invocation is independent; no session state is preserved between calls.
- **Typed I/O contracts** — all inputs and outputs are schema-validated at the protocol layer.
- **Zero external dependencies at call time** — tools execute server-side; the client requires no local installations (e.g., ImageMagick, FFmpeg, OpenSSL).
- **Composable** — multiple tools can be chained within a single agent turn to form complex transformation pipelines.

---

## 3. Problem Statement

In conventional development workflows, auxiliary operations such as string hashing, data serialization, image optimization, or SQL formatting require one of the following approaches:

| Approach | Description | Drawbacks |
|----------|-------------|-----------|
| **External web tools** | Use browser-based utilities (e.g., jwt.io, codebeautify.org) | Context switch, no auditability, copy-paste errors |
| **Ad-hoc scripting** | Write one-off scripts in Python, Bash, etc. | Setup overhead, not reproducible within the conversation |
| **Library installation** | Add a dependency for a single operation | Dependency pollution, version conflicts |
| **Manual computation** | Perform the operation mentally or by hand | Error-prone, unscalable |

DevForge MCP addresses these inefficiencies by providing a **single, uniform invocation interface** accessible directly from within the agent session, eliminating context switching entirely.

Comparative execution model:

```text
# Traditional approach — hash a string
$ python3 -c "import hashlib; print(hashlib.sha256(b'payload').hexdigest())"
# Requires: Python runtime, knowledge of hashlib API, terminal access

# DevForge MCP approach — same operation via agent tool call
Tool: crypto_hash(input="payload", algorithm="sha256", encoding="hex")
# Requires: nothing — executed inline within the conversation
```

---

## 4. Capabilities & Tool Taxonomy

DevForge MCP organizes its tools into 10 functional domains:

| Domain | Key Tools | Primary Use Cases |
|--------|-----------|-------------------|
| **Cryptography** | `crypto_hash`, `crypto_jwt`, `crypto_keygen`, `crypto_password`, `crypto_hmac`, `crypto_mask`, `crypto_random` | Authentication, integrity verification, secret management |
| **Data Processing** | `data_yaml_convert`, `data_json_format`, `data_jsonpath`, `data_schema_validate`, `data_diff`, `data_csv_convert` | Serialization, schema validation, document transformation |
| **HTTP Utilities** | `http_request`, `http_curl_convert`, `http_signed_url`, `http_url_parse`, `http_webhook_replay` | API testing, request generation, URL signing |
| **Frontend** | `frontend_color`, `frontend_breakpoint`, `frontend_css_unit`, `frontend_regex`, `frontend_locale_format`, `frontend_icu_format` | UI development, accessibility, internationalization |
| **Image Processing** | `image_resize`, `image_convert`, `image_crop`, `image_watermark`, `image_srcset`, `image_quality`, `image_placeholder` | Asset optimization, responsive images, format conversion |
| **Code Tooling** | `code_format`, `code_metrics`, `code_template` | Static analysis, formatting, templating |
| **Backend** | `backend_sql_format`, `backend_conn_string`, `backend_env_inspect`, `backend_log_parse`, `backend_mq_payload` | Database tooling, environment validation, observability |
| **Time & Dates** | `time_convert`, `time_cron`, `time_date_range`, `time_diff` | Timestamp normalization, scheduling, range generation |
| **Text Processing** | `text_uuid`, `text_slug`, `text_case`, `text_base64`, `text_url_encode`, `text_escape`, `text_normalize` | String normalization, encoding, identifier generation |
| **Video & Audio** | `video_transcode`, `video_trim`, `video_thumbnail`, `audio_normalize`, `audio_transcode`, `audio_trim` | Media pipeline automation |

**Total registered tools: 70+**

---

## 5. Integration Value Analysis

### 5.1 Capability Shift: Advisory → Operative

The fundamental value of MCP tool integration is the transition from an *advisory* agent model to an *operative* one. Without tool access, a language model can only recommend actions. With DevForge MCP, the agent executes them:

| Operation | Advisory (no tools) | Operative (with DevForge MCP) |
|-----------|--------------------|-----------------------------|
| Password hashing | "Use bcrypt with cost factor 12" | Executes `crypto_password` and returns the hash |
| YAML → JSON conversion | "Use `js-yaml` or an online converter" | Executes `data_yaml_convert` and returns the JSON |
| Image resize | "Use ImageMagick: `convert -resize 640x`" | Executes `image_resize` and returns the output path |
| SQL formatting | "Paste it into sqlformat.org" | Executes `backend_sql_format` and returns formatted SQL |

### 5.2 Tool Composition (Pipeline Pattern)

Tools can be composed within a single agent turn to construct multi-step transformation pipelines without intermediate user intervention:

```text
Pipeline Example: Prepare and sign a deployment artifact

Step 1 → data_json_format     : Validate and normalize configuration JSON
Step 2 → data_yaml_convert    : Serialize configuration as YAML for deployment manifest
Step 3 → crypto_hash          : Compute SHA-256 integrity checksum of the payload
Step 4 → http_signed_url      : Generate HMAC-signed deployment URL with expiry
Step 5 → image_resize         : Generate responsive asset variants for the release

Result: Complete deployment pipeline executed within a single conversation turn.
```

### 5.3 Eliminated Dependencies

The following external tools become optional when DevForge MCP is available in the agent context:

```text
openssl       → crypto_hash, crypto_keygen, crypto_hmac
jq            → data_jsonpath, data_json_format
yq            → data_yaml_convert
ImageMagick   → image_resize, image_convert, image_crop, image_watermark
FFmpeg        → video_transcode, video_trim, audio_normalize
sqlformat     → backend_sql_format
date / GNU    → time_convert, time_diff, time_date_range
uuidgen       → text_uuid
slugify       → text_slug
base64        → text_base64
```

---

## 6. Practical Examples

### 6.1 Cryptographic Hashing — `crypto_hash`

**Tool signature:**
```
crypto_hash(input: string, algorithm: sha256|sha512|md5|sha1, encoding: hex|base64)
```

**Benchmark: digest length by algorithm for input `"Hello World"`**

| Algorithm | Security Level | Output Length | Digest (hex) |
|-----------|---------------|---------------|--------------|
| MD5 | Deprecated | 32 chars | `b10a8db164e0754105b7a99be72e3fe5` |
| SHA-1 | Deprecated | 40 chars | `0a4d55a8d778e5022fab701977c5d840bbc486d0` |
| SHA-256 | Recommended | 64 chars | `a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e` |
| SHA-512 | High security | 128 chars | `2c74fd17edafd80e8447b0d46741ee243b7eb74dd2149a0ab1b9246fb30382f...` |

**Encoding comparison — SHA-256 of `"Hello World"`:**

| Encoding | Output | Length Reduction |
|----------|--------|-----------------|
| `hex` | `a591a6d40bf420404a011733cfb7b190...` | baseline |
| `base64` | `pZGm1Av0IEBKARczz7exkNYsZb8LzaMrV7J32a2fFG4=` | ~33% shorter |

**Important property — case sensitivity:**
```text
SHA-256("Hello World") = a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e
SHA-256("hello world") = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
```
> A single character difference produces a completely uncorrelated digest — this is the avalanche effect, a fundamental property of cryptographic hash functions.

**Note on password storage:** `crypto_hash` is suitable for integrity verification and token generation. For password storage, `crypto_password` with bcrypt or Argon2id must be used instead, as raw SHA-256 without salt is vulnerable to rainbow table attacks.

---

### 6.2 SQL Formatting & Linting — `backend_sql_format`

**Tool signature:**
```
backend_sql_format(sql: string, dialect: postgresql|mysql|sqlite|generic,
                   uppercase_keywords: boolean, indent: string)
```

**Input — unformatted query:**
```sql
select u.id,u.name,u.email,o.total,o.created_at from users u inner join orders o
on u.id=o.user_id where o.total>100 and u.active=true order by o.created_at desc limit 10
```

**Output — formatted (PostgreSQL dialect, uppercase keywords):**
```sql
SELECT u.id, u.name, u.email, o.total, o.created_at
  FROM users u
  INNER JOIN orders o ON u.id = o.user_id
  WHERE o.total > 100 AND u.active = true
  ORDER BY o.created_at DESC
  LIMIT 10
```

**Output — CTE with subquery:**
```sql
WITH monthly_sales AS (
  SELECT date_trunc('month', created_at) AS month,
         user_id,
         SUM(total) AS total_spent
  FROM orders
  WHERE status = 'completed'
  GROUP BY 1, 2
)
SELECT u.name, u.email, ms.month, ms.total_spent,
       (SELECT COUNT(*) FROM orders o2 WHERE o2.user_id = u.id) AS total_orders
  FROM users u
  INNER JOIN monthly_sales ms ON u.id = ms.user_id
  WHERE ms.total_spent > 500
  HAVING COUNT(*) > 2
  ORDER BY ms.total_spent DESC
```

The tool also returns a `warnings` array containing dialect-specific linting diagnostics, suitable for integration into CI/CD pre-commit validation pipelines.

---

### 6.3 Data Format Conversion — `data_yaml_convert`

**Tool signature:**
```
data_yaml_convert(input: string, from: json|yaml, to: json|yaml)
```

**Input (YAML) — Docker Compose service definition:**
```yaml
name: devpixelforge
services:
  api:
    image: golang:1.22
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
  postgres:
    image: postgres:16
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata: {}
```

**Output (JSON):**
```json
{
  "name": "devpixelforge",
  "services": {
    "api": {
      "environment": ["DB_HOST=postgres", "DB_PORT=5432"],
      "image": "golang:1.22",
      "ports": ["8080:8080"]
    },
    "postgres": {
      "image": "postgres:16",
      "volumes": ["pgdata:/var/lib/postgresql/data"]
    }
  },
  "volumes": { "pgdata": {} }
}
```

**Behavioral note:** The conversion engine normalizes object keys to lexicographic order. This is consistent with the JSON specification (RFC 8259), which defines objects as unordered collections. However, consumers relying on insertion-order semantics must handle key ordering explicitly downstream.

**Type fidelity:** Scalar types (`boolean`, `integer`, `float`, `null`) are preserved across both conversion directions.

---

### 6.4 JSONPath Querying — `data_jsonpath`

**Tool signature:**
```
data_jsonpath(json: string, path: string)
```

**Supported syntax:**

| Expression | Semantics |
|------------|-----------|
| `$` | Document root |
| `.field` | Child member access |
| `[N]` | Array index access (0-based) |
| `.*` | All values of an object |
| `[*]` | All elements of an array |

**Test document — product catalog:**
```json
{
  "store": {
    "name": "TechShop",
    "products": [
      { "id": 1, "name": "Laptop",  "price": 1200, "stock": 5  },
      { "id": 2, "name": "Mouse",   "price": 25,   "stock": 50 },
      { "id": 3, "name": "Desk",    "price": 350,  "stock": 10 },
      { "id": 4, "name": "Chair",   "price": 200,  "stock": 8  }
    ],
    "owner": { "name": "Carlos", "email": "carlos@techshop.com" }
  }
}
```

**Query results:**

| JSONPath Expression | Result |
|--------------------|--------|
| `$.store.products[0].name` | `"Laptop"` |
| `$.store.products[*].name` | `["Laptop", "Mouse", "Desk", "Chair"]` |
| `$.store.products[*].price` | `[1200, 25, 350, 200]` |
| `$.store.owner.*` | `["Carlos", "carlos@techshop.com"]` |

JSONPath evaluation eliminates the need to deserialize and traverse full document structures when only a subset of fields is required — a common pattern in API response processing and log analysis.

---

### 6.5 Temporal Sequence Generation — `time_date_range`

**Tool signature:**
```
time_date_range(start: YYYY-MM-DD, end: YYYY-MM-DD,
                step: day|week|month, format: iso8601|unix|human)
```

**Parameters used:** Q1 2026, weekly step, Unix timestamp output.

**Output — 13 weekly intervals (partial):**

| Week | Label | Unix Timestamp |
|------|-------|----------------|
| 1 | Jan 1, 2026 | `1767225600` |
| 2 | Jan 8, 2026 | `1767830400` |
| 3 | Jan 15, 2026 | `1768435200` |
| 4 | Jan 22, 2026 | `1769040000` |
| 5 | Jan 29, 2026 | `1769644800` |
| ... | ... | ... |
| 13 | Mar 26, 2026 | `1774483200` |

**Inter-interval invariant:**

```
Δt = 1767830400 − 1767225600 = 604,800 seconds
604,800 = 7 days × 24 hours × 60 minutes × 60 seconds  ✓
```

Unix timestamps enable direct arithmetic operations on temporal sequences without requiring date/time library dependencies in the consuming application.

**Maximum output cardinality:** 1,000 dates per invocation.

---

## 7. Role-Based Applicability Matrix

| Engineering Role | High-Value Tool Subset | Primary Benefit |
|-----------------|------------------------|-----------------|
| **Backend Engineer** | `backend_sql_format`, `backend_conn_string`, `backend_log_parse`, `backend_env_inspect`, `crypto_jwt`, `crypto_hmac` | Database tooling, observability, authentication primitives |
| **Frontend Engineer** | `frontend_color`, `frontend_breakpoint`, `frontend_css_unit`, `frontend_regex`, `frontend_locale_format` | UI consistency, accessibility compliance (WCAG 2.1), i18n |
| **DevOps / SRE** | `backend_env_inspect`, `http_signed_url`, `backend_log_parse`, `time_cron`, `backend_mq_payload` | Infrastructure validation, scheduling, message queue tooling |
| **Full-Stack Engineer** | All of the above + `image_resize`, `video_transcode`, `audio_normalize` | Complete media pipeline without external binary dependencies |
| **Any Role** | `text_uuid`, `crypto_hash`, `text_base64`, `time_convert`, `data_yaml_convert`, `file_diff` | Universal developer utilities — zero external tooling required |

---

## 8. Conclusions

DevForge MCP represents a paradigm shift in how AI-assisted development tools interact with the software engineering workflow. By exposing a comprehensive set of deterministic, typed developer utilities through the Model Context Protocol, it eliminates the primary friction point in AI-assisted development: the gap between *recommendation* and *execution*.

The key technical properties that differentiate this approach from conventional tooling are:

1. **Uniform invocation interface** — all 70+ tools share the same call semantics, reducing cognitive overhead.
2. **No runtime dependencies** — operations execute server-side; the developer environment requires no additional installations.
3. **Composability** — tools can be chained within a single agent turn, enabling multi-step pipelines without user intervention.
4. **Auditability** — all tool inputs and outputs are visible in the conversation context, providing a natural audit trail.

The net effect is a measurable reduction in context-switching overhead and an expansion of the agent's operative surface from *advisory* to *collaborative execution* — a distinction with direct implications for developer productivity and workflow continuity.

---

---

## References

| # | Resource | URL |
|---|----------|-----|
| 1 | **DevForge MCP** — MCP server source repository | [github.com/GustavoGutierrez/devforge](https://github.com/GustavoGutierrez/devforge) |
| 2 | **DevForge MCP** — Official documentation | [mintlify.com/GustavoGutierrez/devforge](https://www.mintlify.com/GustavoGutierrez/devforge) |
| 3 | **DevPixelForge (dpf)** — Native image processing engine (Rust + Go bridge) | [github.com/GustavoGutierrez/devpixelforge](https://github.com/GustavoGutierrez/devpixelforge) |
| 4 | **DevPixelForge (dpf)** — Official documentation | [mintlify.com/GustavoGutierrez/devpixelforge](https://www.mintlify.com/GustavoGutierrez/devpixelforge) |
| 5 | Model Context Protocol specification | [modelcontextprotocol.io](https://modelcontextprotocol.io) |
| 6 | RFC 8259 — The JavaScript Object Notation (JSON) Data Interchange Format | [datatracker.ietf.org/doc/html/rfc8259](https://datatracker.ietf.org/doc/html/rfc8259) |
| 7 | JSONPath — RFC 9535 | [datatracker.ietf.org/doc/html/rfc9535](https://datatracker.ietf.org/doc/html/rfc9535) |
| 8 | NIST FIPS 180-4 — Secure Hash Standard (SHA-256, SHA-512) | [csrc.nist.gov/publications/detail/fips/180/4/final](https://csrc.nist.gov/publications/detail/fips/180/4/final) |

### Relationship between DevForge MCP and DevPixelForge (dpf)

**DevPixelForge (dpf)** is the native binary engine that powers the image and media processing tools exposed by DevForge MCP. It is implemented in **Rust** with a **Go FFI bridge**, compiled with LTO (Link-Time Optimization) and `codegen-units=1` for maximum performance.

```text
DevForge MCP (tool interface)
        │
        ▼
  devforge_optimize_images
  devforge_image_resize
  devforge_image_convert
  devforge_image_crop
  devforge_image_watermark
  devforge_image_srcset
  ...
        │
        ▼
  DevPixelForge / dpf  (Rust binary — native execution)
  github.com/GustavoGutierrez/devpixelforge
```

The MCP server invokes the `dpf` binary via the Go bridge layer, passing structured JSON job payloads through stdin/stdout. This architecture decouples the tool interface from the processing engine, allowing the Rust binary to be updated independently without modifying the MCP server contract.

```bash
# Direct dpf invocation (used internally by DevForge MCP)
dpf process --job '{"operation":"resize","input":"img.png","output_dir":"/tmp","widths":[320,640]}'
```

---

*End of document — DevForge MCP Technical Reference v1.0.0*  
*Author: Ing. Gustavo Gutiérrez — Bogotá, Colombia*
