# datetime Tools

Date and time utilities: format conversion, difference calculation, cron scheduling, and date range generation.

---

## `time_convert`

Convert a timestamp between different formats. Auto-detects the input format when `from_format` is omitted.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `input` | string | ✅ | — | Timestamp to convert |
| `from_format` | string | | `auto` | Source format: `unix` \| `unix_ms` \| `iso8601` \| `rfc3339` \| `human` \| `auto` |
| `to_format` | string | | `rfc3339` | Target format: `unix` \| `unix_ms` \| `iso8601` \| `rfc3339` \| `human` |
| `timezone` | string | | `UTC` | IANA timezone name for output (e.g. `America/New_York`) |

### Format Reference

| Format | Example |
|--------|---------|
| `unix` | `1710498600` |
| `unix_ms` | `1710498600000` |
| `iso8601` | `2024-03-15T10:30:00+00:00` |
| `rfc3339` | `2024-03-15T10:30:00Z` |
| `human` | `Mar 15, 2024 10:30:00 UTC` |

### Auto-detection Rules

- Numeric strings `> 1_000_000_000_000` → `unix_ms`
- Other numeric strings → `unix`
- All other strings: tries rfc3339, iso8601, human layouts in order

### Output

```json
{
  "result": "2024-03-15T10:30:00Z",
  "from_format": "unix",
  "to_format": "rfc3339",
  "timezone": "UTC"
}
```

### Examples

```json
// Unix timestamp → RFC3339
{ "input": "1710498600", "from_format": "unix", "to_format": "rfc3339", "timezone": "UTC" }

// ISO8601 → Unix milliseconds
{ "input": "2024-03-15T10:30:00Z", "to_format": "unix_ms" }

// With timezone conversion
{ "input": "2024-03-15T10:30:00Z", "to_format": "human", "timezone": "America/New_York" }
```

---

## `time_diff`

Calculate the difference between two timestamps, or add/subtract a duration from a timestamp.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `start` | string | ✅ | — | Start timestamp (ISO8601 or RFC3339) |
| `end` | string | | — | End timestamp — required when `operation=diff` |
| `operation` | string | | `diff` | `diff` \| `add` \| `subtract` |
| `duration` | string | | — | Duration for `add`/`subtract`: Go format (`2h30m`) or English (`3 days`) |
| `unit` | string | | `auto` | Preferred unit: `auto` \| `seconds` \| `minutes` \| `hours` \| `days` \| `weeks` |

### Duration Formats

| Format | Example | Description |
|--------|---------|-------------|
| Go duration | `2h30m`, `90s`, `1h30m45s` | Standard Go `time.Duration` string |
| English | `3 days`, `1 week`, `30 minutes` | `<n> <unit>` where unit is singular |

Supported English units: `second`, `minute`, `hour`, `day`, `week` (plurals accepted).

### Output — `diff`

```json
{
  "seconds": 86400.0,
  "minutes": 1440.0,
  "hours": 24.0,
  "days": 1.0,
  "human": "1 day"
}
```

### Output — `add` / `subtract`

```json
{
  "result": "2024-01-03T00:00:00Z"
}
```

### Examples

```json
// Diff between two timestamps
{
  "start": "2024-01-01T00:00:00Z",
  "end": "2024-01-03T12:00:00Z",
  "operation": "diff"
}

// Add 2 hours and 30 minutes
{
  "start": "2024-01-01T00:00:00Z",
  "operation": "add",
  "duration": "2h30m"
}

// Subtract 3 days
{
  "start": "2024-01-10T00:00:00Z",
  "operation": "subtract",
  "duration": "3 days"
}
```

---

## `time_cron`

Describe a cron expression in plain English, or compute the next N execution times.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `expression` | string | ✅ | — | Cron expression (5 or 6 fields) |
| `operation` | string | | `describe` | `describe` \| `next` |
| `count` | number | | `5` | Number of next execution times to compute |
| `from` | string | | now | Reference datetime for `next` computation |

### Cron Field Formats

**5 fields**: `<minute> <hour> <dom> <month> <dow>`

**6 fields**: `<second> <minute> <hour> <dom> <month> <dow>`

Standard cron syntax including:
- `*` — every unit
- `*/N` — every N units
- `N` — specific value
- `N-M` — range
- `N,M` — list
- Named days/months: `MON`, `TUE`, ..., `JAN`, `FEB`, ...

### Output — `describe`

```json
{
  "description": "At 9:00 AM, every day",
  "valid": true
}
```

Invalid expression:
```json
{
  "valid": false,
  "error": "expected exactly 5 or 6 fields, found 3: not a cron"
}
```

### Output — `next`

```json
{
  "next": [
    "2024-01-01T09:00:00Z",
    "2024-01-02T09:00:00Z",
    "2024-01-03T09:00:00Z"
  ],
  "count": 3
}
```

### Examples

```json
// Describe a daily job at 9 AM
{ "expression": "0 9 * * *", "operation": "describe" }

// Next 5 runs of an every-15-minutes job
{ "expression": "*/15 * * * *", "operation": "next", "count": 5 }

// Next runs from a specific reference time
{
  "expression": "0 9 * * MON-FRI",
  "operation": "next",
  "count": 3,
  "from": "2024-01-01T00:00:00Z"
}
```

---

## `time_date_range`

Generate a list of dates between two ISO8601 dates with a configurable step.

**Constraint**: Result is capped at **1000 dates**. Returns an error if the range would exceed this limit.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `start` | string | ✅ | — | Start date (`YYYY-MM-DD`) |
| `end` | string | ✅ | — | End date (`YYYY-MM-DD`) |
| `step` | string | | `day` | Step size: `day` \| `week` \| `month` |
| `format` | string | | `iso8601` | Output format: `iso8601` \| `unix` \| `human` |

### Output

```json
{
  "dates": [
    "2024-01-01",
    "2024-01-02",
    "2024-01-03"
  ],
  "count": 3
}
```

### Examples

```json
// All days in January 2024
{ "start": "2024-01-01", "end": "2024-01-31", "step": "day" }

// Weekly milestones for a quarter
{ "start": "2024-01-01", "end": "2024-03-31", "step": "week" }

// Monthly billing dates
{ "start": "2024-01-01", "end": "2024-12-01", "step": "month" }

// Unix timestamps for days
{ "start": "2024-01-01", "end": "2024-01-07", "format": "unix" }

// Human-readable dates
{ "start": "2024-01-01", "end": "2024-01-05", "format": "human" }
```

### Error Cases

| Condition | Error |
|-----------|-------|
| `start` missing | `"start is required"` |
| `end` missing | `"end is required"` |
| `end` before `start` | `"end must be on or after start"` |
| Range > 1000 dates | `"date range would produce more than 1000 dates"` |
| Invalid format | `"unsupported format ..."` |
