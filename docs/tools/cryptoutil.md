# cryptoutil — Security & Cryptography Tools

The `cryptoutil` package exposes 7 MCP tools for common cryptographic operations: hashing, HMAC, JWT management, password hashing, key pair generation, random value generation, and secret masking.

All tools are stateless and require no external dependencies beyond the Go standard library and `golang.org/x/crypto`.

---

## Tools

### `crypto_hash`

Hash a string using a standard algorithm.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `input` | string | ✅ | — | String to hash |
| `algorithm` | string | | `sha256` | `sha256` \| `sha512` \| `md5` \| `sha1` |
| `encoding` | string | | `hex` | `hex` \| `base64` |

**Output**

```json
{
  "hash": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c...",
  "algorithm": "sha256",
  "encoding": "hex"
}
```

**Example**

```json
{
  "input": "hello",
  "algorithm": "sha256",
  "encoding": "hex"
}
```

---

### `crypto_hmac`

Compute an HMAC signature for message authentication.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `message` | string | ✅ | — | Message to authenticate |
| `key` | string | ✅ | — | HMAC secret key |
| `algorithm` | string | | `sha256` | `sha256` \| `sha512` |
| `encoding` | string | | `hex` | `hex` \| `base64` |

**Output**

```json
{
  "hmac": "b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad",
  "algorithm": "sha256"
}
```

---

### `crypto_jwt`

Generate, decode, or verify JWT tokens (HS256/HS512). Implemented without external JWT libraries using standard `crypto/hmac`.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `operation` | string | ✅ | — | `decode` \| `verify` \| `generate` |
| `token` | string | for decode/verify | — | JWT token string |
| `secret` | string | for verify/generate | — | HMAC secret key |
| `payload` | string | | `{}` | JSON payload object (for generate) |
| `expiry_seconds` | int | | `3600` | Token expiry in seconds (for generate) |
| `algorithm` | string | | `HS256` | `HS256` \| `HS512` |

**Output — decode**

```json
{
  "header": { "alg": "HS256", "typ": "JWT" },
  "payload": { "sub": "user123", "iat": 1716000000, "exp": 1716003600 },
  "signature": "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
  "expired": false
}
```

**Output — verify**

```json
{
  "valid": true,
  "expired": false,
  "claims": { "sub": "user123", "role": "admin" }
}
```

**Output — generate**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.abc123"
}
```

**Notes**

- `iat` and `exp` claims are automatically added on generate.
- `verify` returns `valid: false` if the signature does not match **or** the token is expired.
- Does not use any external JWT library — pure HMAC over base64url-encoded header.payload.

---

### `crypto_password`

Hash or verify passwords using industry-standard algorithms.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `password` | string | ✅ | — | Password to hash or verify |
| `operation` | string | ✅ | — | `hash` \| `verify` |
| `algorithm` | string | | `bcrypt` | `bcrypt` \| `argon2id` |
| `hash` | string | for verify | — | Previously computed hash |
| `cost` | int | | `12` | bcrypt work factor (4–31) |

**Output — hash**

```json
{ "hash": "$2a$12$..." }
```

**Output — verify**

```json
{ "valid": true }
```

**Argon2id parameters** (not configurable, uses OWASP recommended defaults):
- `m=65536` (64 MB memory)
- `t=1` (1 iteration)
- `p=4` (4 parallelism)
- Key length: 32 bytes

The argon2id hash is stored in PHC format: `$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`.

---

### `crypto_keygen`

Generate asymmetric key pairs in PEM or JWK format.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `key_type` | string | ✅ | — | `rsa` \| `ec` \| `ed25519` |
| `bits` | int | | `2048` | RSA key size: `2048` or `4096` |
| `curve` | string | | `P-256` | EC curve: `P-256` \| `P-384` |
| `format` | string | | `pem` | `pem` \| `jwk` |

**Output**

```json
{
  "private_key": "-----BEGIN PRIVATE KEY-----\n...",
  "public_key": "-----BEGIN PUBLIC KEY-----\n...",
  "key_type": "rsa"
}
```

For JWK format, `private_key` and `public_key` are JSON strings:

```json
{
  "private_key": "{\"kty\":\"RSA\",\"n\":\"...\",\"e\":\"...\",\"d\":\"...\"}",
  "public_key": "{\"kty\":\"RSA\",\"n\":\"...\",\"e\":\"...\"}",
  "key_type": "rsa"
}
```

**Key type details**

| Type | PEM format | JWK kty |
|------|-----------|---------|
| RSA | `PRIVATE KEY` (PKCS#8) + `PUBLIC KEY` (PKIX) | `RSA` |
| EC | `EC PRIVATE KEY` + `PUBLIC KEY` (PKIX) | `EC` with `crv` |
| Ed25519 | `PRIVATE KEY` (PKCS#8) + `PUBLIC KEY` (PKIX) | `OKP` with `crv: Ed25519` |

---

### `crypto_random`

Generate cryptographically secure random values using `crypto/rand`.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `kind` | string | ✅ | — | `token` \| `bytes` \| `otp` |
| `length` | int | | `32` (token/bytes), `6` (otp) | Length in bytes or digits |
| `encoding` | string | | `hex` | `hex` \| `base64` \| `base64url` (ignored for otp) |

**Output**

```json
{ "value": "a3f7bc9e1d4..." }
```

**Kind details**

| Kind | Description | Example output |
|------|-------------|----------------|
| `token` | Random bytes encoded as string | `"a3f7bc9e..."` (hex) |
| `bytes` | Same as token | `"a3f7bc9e..."` (hex) |
| `otp` | Numeric digits only | `"492817"` |

---

### `crypto_mask`

Scan text and redact sensitive data using regex patterns.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `text` | string | ✅ | — | Text to scan and redact |
| `patterns` | string[] | | `["all"]` | `api_key` \| `password` \| `email` \| `credit_card` \| `jwt` \| `all` |
| `replacement` | string | | `[REDACTED]` | Replacement string for matched secrets |

**Output**

```json
{
  "result": "Use token [REDACTED] to authenticate",
  "redacted_count": 1
}
```

**Pattern details**

| Pattern | Detects |
|---------|---------|
| `api_key` | Tokens starting with `sk-`, `pk-`, `api_`, `key_` followed by 20+ characters |
| `password` | Key-value pairs like `password=hunter2`, `passwd: "foo"` |
| `email` | Standard email addresses |
| `credit_card` | 13–19 consecutive digits (with optional separators) |
| `jwt` | Three-part base64url tokens starting with `eyJ` |
| `all` | Activates all patterns above |

**Notes**

- Patterns are applied sequentially; redacted_count reflects total matches across all patterns.
- The `replacement` string itself is never re-scanned for patterns.
- Use `patterns: ["email", "api_key"]` to activate only specific patterns.

---

## Error Format

All tools return `{"error": "message"}` on invalid input. The server never panics.

Common error cases:

| Tool | Error condition |
|------|----------------|
| `crypto_hash` | Empty `input`; unknown `algorithm` |
| `crypto_hmac` | Empty `message` or `key`; unknown `algorithm` |
| `crypto_jwt` | Missing `operation`; invalid token format; missing `secret` for verify/generate |
| `crypto_password` | Empty `password`; missing `hash` for verify; unknown `algorithm` |
| `crypto_keygen` | Missing `key_type`; invalid `bits` for RSA; unknown `curve` for EC |
| `crypto_random` | Missing `kind` |
| `crypto_mask` | Empty `text` |

---

## Package Structure

```
internal/tools/cryptoutil/
├── cryptoutil.go        — Tool implementations (Hash, HMAC, JWT, Password, Keygen, Random, Mask)
└── cryptoutil_test.go   — Table-driven tests (package cryptoutil_test)

cmd/devforge-mcp/
└── register_cryptoutil.go  — MCP tool registration (package main)

docs/tools/
└── cryptoutil.md        — This file
```
