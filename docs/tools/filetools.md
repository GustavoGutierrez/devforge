# filetools — File & Archive Tools

Group 6 of the Developer Utilities Toolkit. Provides file integrity, archiving, diffing, line-ending normalisation, and hex inspection tools.

All tools are **stateless** and safe for concurrent use. No database writes or external network calls are made.

---

## `file_checksum`

Calculate the MD5, SHA-256, or SHA-512 checksum of a file.

The file is read as a stream — it is never fully loaded into memory — making this safe for very large files.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `path` | string | yes | — | Absolute or relative path to the target file |
| `algorithm` | string | no | `sha256` | Hash algorithm: `md5` \| `sha256` \| `sha512` |

### Return schema

```json
{
  "checksum": "<hex string>",
  "algorithm": "sha256",
  "path": "/path/to/file",
  "size_bytes": 1024
}
```

### Error responses

| Condition | Error message |
|-----------|---------------|
| `path` is empty | `"path is required"` |
| File does not exist | `"cannot open file: ..."` |
| Unknown algorithm | `"unknown algorithm: must be md5, sha256, or sha512"` |

### Example

```json
{
  "path": "/home/user/downloads/archive.tar.gz",
  "algorithm": "sha256"
}
```

```json
{
  "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "algorithm": "sha256",
  "path": "/home/user/downloads/archive.tar.gz",
  "size_bytes": 204800
}
```

---

## `file_archive`

Create or extract `zip` or `tar.gz` archives with optional glob-based exclusion patterns.

Path traversal (zip-slip) is prevented: extracted entries are always confined to the destination directory.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `operation` | string | yes | — | `create` \| `extract` |
| `format` | string | no | `zip` | Archive format: `zip` \| `tar.gz` |
| `source` | string | create only | — | Source file or directory to archive |
| `output` | string | create only | — | Output archive path |
| `archive` | string | extract only | — | Archive file to extract |
| `dest` | string | extract only | — | Destination directory |
| `exclude` | string[] | no | `[]` | Glob patterns to exclude (e.g. `["*.log", "tmp/"]`) |

### Return schema — create

```json
{
  "archive": "/path/to/output.zip",
  "files_added": 42,
  "size_bytes": 102400
}
```

### Return schema — extract

```json
{
  "dest": "/path/to/destination",
  "files_extracted": 42
}
```

### Error responses

| Condition | Error message |
|-----------|---------------|
| Unknown `operation` | `"unknown operation: must be create or extract"` |
| Unknown `format` | `"unknown format: must be zip or tar.gz"` |
| Missing `source` for create | `"source is required for create operation"` |
| Missing `output` for create | `"output is required for create operation"` |
| Missing `archive` for extract | `"archive is required for extract operation"` |
| Missing `dest` for extract | `"dest is required for extract operation"` |
| Archive creation fails | `"archive creation failed: ..."` |
| Zip-slip detected | `"illegal file path: ... would escape destination"` |

### Examples

**Create a zip archive, excluding log files:**

```json
{
  "operation": "create",
  "format": "zip",
  "source": "./dist",
  "output": "./release/dist.zip",
  "exclude": ["*.log", "*.tmp"]
}
```

```json
{
  "archive": "./release/dist.zip",
  "files_added": 18,
  "size_bytes": 51200
}
```

**Extract a tar.gz archive:**

```json
{
  "operation": "extract",
  "format": "tar.gz",
  "archive": "./backups/2026-03-01.tar.gz",
  "dest": "./restored"
}
```

```json
{
  "dest": "./restored",
  "files_extracted": 73
}
```

---

## `file_diff`

Generate a unified diff between two files or two text strings.

Uses an LCS-based diff algorithm (Myers-style DP) with no external dependencies. Output follows the standard unified diff format (`--- a\n+++ b\n@@ ... @@`).

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `a` | string | yes | — | First file path (`file` mode) or text content (`text` mode) |
| `b` | string | yes | — | Second file path (`file` mode) or text content (`text` mode) |
| `mode` | string | no | `file` | `file` \| `text` |
| `context_lines` | int | no | `3` | Number of unchanged context lines around each changed hunk |

### Return schema

```json
{
  "diff": "--- a.txt\n+++ b.txt\n@@ -1,3 +1,4 @@\n ...",
  "additions": 2,
  "deletions": 1
}
```

When the two inputs are identical, `diff` is an empty string and both counters are `0`.

### Error responses

| Condition | Error message |
|-----------|---------------|
| Unknown `mode` | `"unknown mode: must be file or text"` |
| File not readable | `"cannot read file a: ..."` / `"cannot read file b: ..."` |
| Missing file paths in file mode | `"a and b file paths are required"` |

### Example

```json
{
  "a": "./before.sql",
  "b": "./after.sql",
  "mode": "file",
  "context_lines": 2
}
```

```json
{
  "diff": "--- ./before.sql\n+++ ./after.sql\n@@ -3,5 +3,6 @@\n  SELECT\n -  name,\n +  name AS user_name,\n +  email,\n   FROM users;\n",
  "additions": 2,
  "deletions": 1
}
```

---

## `file_line_endings`

Detect, normalize, or convert line endings (LF / CRLF) in a file or text string.

- **detect**: counts LF-only and CRLF occurrences and reports the dominant style.
- **normalize** / **convert**: rewrites all endings to the specified `target`.
- In `text` mode, normalized text is returned inline; in `file` mode, the file is written to `output` (or overwritten in place if `output` is omitted).

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `input` | string | yes | — | File path (`file` mode) or raw text (`text` mode) |
| `mode` | string | no | `file` | `file` \| `text` |
| `operation` | string | no | `detect` | `detect` \| `normalize` \| `convert` |
| `target` | string | no | `lf` | Target line ending for normalize/convert: `lf` \| `crlf` |
| `output` | string | no | *(overwrite)* | Output file path (file mode only; omit to overwrite input) |

### Return schema — detect

```json
{
  "line_ending": "crlf",
  "lf_count": 0,
  "crlf_count": 42
}
```

`line_ending` is one of `"lf"`, `"crlf"`, or `"mixed"`.

### Return schema — text mode normalize/convert

```json
{
  "result": "line1\nline2\n"
}
```

### Return schema — file mode normalize/convert

```json
{
  "output": "/path/to/file.txt",
  "lines_converted": 42
}
```

### Error responses

| Condition | Error message |
|-----------|---------------|
| Missing `input` in file mode | `"input file path is required"` |
| File not readable | `"cannot read file: ..."` |
| Unknown `mode` | `"unknown mode: must be file or text"` |
| Unknown `operation` | `"unknown operation: must be normalize, detect, or convert"` |
| Unknown `target` | `"unknown target: must be lf or crlf"` |

### Examples

**Detect line endings:**

```json
{ "input": "./legacy/windows_file.txt", "mode": "file", "operation": "detect" }
```

```json
{ "line_ending": "crlf", "lf_count": 0, "crlf_count": 87 }
```

**Normalize CRLF → LF in-place:**

```json
{
  "input": "./legacy/windows_file.txt",
  "mode": "file",
  "operation": "normalize",
  "target": "lf"
}
```

```json
{ "output": "./legacy/windows_file.txt", "lines_converted": 87 }
```

---

## `file_hex_view`

Display binary file content or base64-encoded bytes as a formatted hex + ASCII dump table.

Output uses the standard hex dump format:

```
00000000  48 65 6c 6c 6f 20 57 6f  72 6c 64 0a 00 01 02 03  |Hello World.....|
```

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `input` | string | yes | — | File path (`file` mode) or base64-encoded bytes (`base64` mode) |
| `mode` | string | no | `file` | `file` \| `base64` |
| `offset` | int | no | `0` | Byte offset to start reading from |
| `length` | int | no | `256` | Number of bytes to display |
| `width` | int | no | `16` | Bytes per row |

### Return schema

```json
{
  "hex_view": "00000000  48 65 6c 6c 6f ...\n",
  "offset": 0,
  "bytes_shown": 16,
  "total_bytes": 1024
}
```

- `offset` — the byte position where the view starts.
- `bytes_shown` — number of bytes actually displayed (may be less than `length` near end of file).
- `total_bytes` — total size of the source (file size or decoded base64 length).

### Error responses

| Condition | Error message |
|-----------|---------------|
| Missing `input` | `"input file path is required"` / `"input base64 data is required"` |
| File does not exist | `"cannot open file: ..."` |
| Offset beyond file size | `"offset N exceeds file size M"` |
| Invalid base64 | `"base64 decode failed: ..."` |
| Unknown `mode` | `"unknown mode: must be file or base64"` |

### Examples

**View first 64 bytes of a binary file:**

```json
{
  "input": "./firmware.bin",
  "mode": "file",
  "offset": 0,
  "length": 64,
  "width": 16
}
```

```json
{
  "hex_view": "00000000  7f 45 4c 46 02 01 01 00  00 00 00 00 00 00 00 00  |.ELF............|\n00000010  02 00 3e 00 01 00 00 00  ...\n",
  "offset": 0,
  "bytes_shown": 64,
  "total_bytes": 131072
}
```

**Inspect base64-encoded data:**

```json
{
  "input": "SGVsbG8gV29ybGQ=",
  "mode": "base64",
  "width": 8
}
```

```json
{
  "hex_view": "00000000  48 65 6c 6c  6f 20 57 6f  |Hello Wo|\n00000008  72 6c 64     ...\n",
  "offset": 0,
  "bytes_shown": 11,
  "total_bytes": 11
}
```
