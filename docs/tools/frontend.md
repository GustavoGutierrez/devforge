# Frontend Utilities — Tool Reference

> Group 7 · Package `dev-forge-mcp/internal/tools/frontend`

Six stateless tools for frontend development: color conversion, CSS unit conversion, responsive breakpoints, regex evaluation, locale-aware formatting, and ICU message formatting.

---

## `frontend_color`

Convert a color between HEX, RGB, HSL, HSLA, and RGBA formats. Optionally compute the WCAG 2.1 contrast ratio against a second color.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `color` | string | ✓ | — | Source color: `#RRGGBB`, `#RGB`, `rgb(r,g,b)`, or `hsl(h,s%,l%)` |
| `to` | string | | `hex` | Target format: `hex` \| `rgb` \| `hsl` \| `hsla` \| `rgba` |
| `alpha` | float | | `1.0` | Alpha channel 0.0–1.0 for `rgba`/`hsla` output |
| `against` | string | | — | Second color for WCAG contrast computation (same format as `color`) |

### Return Schema

```json
{
  "result": "<converted color string>",
  "contrast_ratio": 4.52,
  "wcag_aa": true,
  "wcag_aaa": false
}
```

`contrast_ratio`, `wcag_aa`, and `wcag_aaa` are only present when `against` is provided.

- **WCAG AA** (normal text): contrast ratio ≥ 4.5:1
- **WCAG AAA** (enhanced): contrast ratio ≥ 7.0:1
- Contrast ratio formula: `(L1 + 0.05) / (L2 + 0.05)` where L1 > L2

### Examples

```json
// Convert hex to HSL
{ "color": "#ff0000", "to": "hsl" }
// → { "result": "hsl(0, 100%, 50%)" }

// Check contrast ratio (black on white)
{ "color": "#000000", "to": "hex", "against": "#ffffff" }
// → { "result": "#000000", "contrast_ratio": 21, "wcag_aa": true, "wcag_aaa": true }

// RGBA with transparency
{ "color": "#3b82f6", "to": "rgba", "alpha": 0.8 }
// → { "result": "rgba(59, 130, 246, 0.80)" }
```

---

## `frontend_css_unit`

Convert CSS values between `px`, `rem`, `em`, `percent`, `vw`, and `vh`.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `value` | float | ✓ | — | Source numeric value |
| `from` | string | ✓ | — | Source unit: `px` \| `rem` \| `em` \| `percent` \| `vw` \| `vh` |
| `to` | string | ✓ | — | Target unit (same options) |
| `base_font_size` | float | | `16` | Root `<html>` font size in px (for `rem`) |
| `viewport_width` | float | | `1440` | Viewport width in px (for `vw`) |
| `viewport_height` | float | | `900` | Viewport height in px (for `vh`) |
| `parent_size` | float | | `16` | Parent element size in px (for `em` and `percent`) |

### Return Schema

```json
{
  "result": 1.5,
  "from": "px",
  "to": "rem",
  "formatted": "1.5rem"
}
```

### Examples

```json
// 24px to rem (base 16)
{ "value": 24, "from": "px", "to": "rem", "base_font_size": 16 }
// → { "result": 1.5, "from": "px", "to": "rem", "formatted": "1.5rem" }

// 50% of 800px parent to px
{ "value": 50, "from": "percent", "to": "px", "parent_size": 800 }
// → { "result": 400, "from": "percent", "to": "px", "formatted": "400px" }
```

---

## `frontend_breakpoint`

Identify the responsive breakpoint for a viewport width and generate the corresponding CSS `@media` query.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `width` | int | ✓ | — | Viewport width in pixels |
| `system` | string | | `tailwind` | `tailwind` \| `bootstrap` \| `custom` |
| `custom_breakpoints` | object | | — | `{ name: minWidthPx }` pairs — required when `system=custom` |
| `generate_query` | bool | | `true` | Include generated `@media` query in response |

### Built-in Breakpoints

**Tailwind v4**

| Name | Min Width |
|------|-----------|
| xs   | 0 px      |
| sm   | 640 px    |
| md   | 768 px    |
| lg   | 1024 px   |
| xl   | 1280 px   |
| 2xl  | 1536 px   |

**Bootstrap 5**

| Name | Min Width |
|------|-----------|
| xs   | 0 px      |
| sm   | 576 px    |
| md   | 768 px    |
| lg   | 992 px    |
| xl   | 1200 px   |
| xxl  | 1400 px   |

### Return Schema

```json
{
  "breakpoint": "md",
  "min_width": 768,
  "max_width": 1023,
  "media_query": "@media (min-width: 768px) { ... }"
}
```

`max_width` is `null` for the largest breakpoint. `media_query` is omitted when `generate_query=false`.

### Examples

```json
// Tailwind at 900px
{ "width": 900, "system": "tailwind" }
// → { "breakpoint": "md", "min_width": 768, "max_width": 1023, "media_query": "@media (min-width: 768px) { ... }" }

// Custom breakpoints
{ "width": 900, "system": "custom", "custom_breakpoints": { "mobile": 0, "tablet": 600, "desktop": 1024 } }
// → { "breakpoint": "tablet", "min_width": 600, "max_width": 1023, ... }
```

---

## `frontend_regex`

Test, match, or replace using a regular expression. Uses Go's `regexp` package (RE2 syntax).

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `pattern` | string | ✓ | — | Regular expression (no delimiters) |
| `input` | string | ✓ | — | Input string |
| `flags` | string | | — | `i` (case-insensitive), `m` (multiline), `g` (global/all). Combine: `"ig"` |
| `operation` | string | | `test` | `test` \| `match` \| `replace` |
| `replacement` | string | | `""` | Replacement string for `replace` operation. Supports `$1`, `$2` group references. |

> **Note**: RE2 syntax does not support lookaheads or backreferences. See [RE2 syntax](https://github.com/google/re2/wiki/Syntax).

### Return Schemas

**test**
```json
{ "matches": true, "count": 3 }
```

**match**
```json
{
  "matches": [
    { "full": "user@example", "groups": ["user", "example"], "index": 0 }
  ]
}
```

**replace**
```json
{ "result": "price: XXX USD", "count": 1 }
```

### Examples

```json
// Test a digit pattern
{ "pattern": "\\d+", "input": "order #12345", "operation": "test" }
// → { "matches": true, "count": 1 }

// Extract email parts
{ "pattern": "(\\w+)@(\\w+\\.\\w+)", "input": "send to user@example.com", "operation": "match" }
// → { "matches": [{ "full": "user@example.com", "groups": ["user", "example.com"], "index": 8 }] }

// Replace all digits globally
{ "pattern": "\\d+", "input": "v1.2.3", "operation": "replace", "replacement": "X", "flags": "g" }
// → { "result": "vX.X.X", "count": 3 }
```

---

## `frontend_locale_format`

Format numbers, dates, times, currencies, and percentages using IETF locale conventions. Best-effort implementation for 8 common locales.

### Supported Locales

| Locale | Region | Separators |
|--------|--------|------------|
| `en-US` | US English | `,` thousands, `.` decimal |
| `en-GB` | UK English | `,` thousands, `.` decimal |
| `de-DE` | German | `.` thousands, `,` decimal |
| `fr-FR` | French | `\u202F` thousands, `,` decimal |
| `es-ES` | Spanish | `.` thousands, `,` decimal |
| `pt-BR` | Brazilian Portuguese | `.` thousands, `,` decimal |
| `ja-JP` | Japanese | `,` thousands, `.` decimal |
| `zh-CN` | Simplified Chinese | `,` thousands, `.` decimal |

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `value` | string | ✓ | — | Number (as string) or ISO 8601 date/datetime |
| `kind` | string | ✓ | — | `number` \| `currency` \| `date` \| `time` \| `datetime` \| `percent` |
| `locale` | string | | `en-US` | IETF locale tag |
| `currency` | string | | — | ISO 4217 code (`USD`, `EUR`, `GBP`, …) — required for `kind=currency` |
| `options` | object | | — | `decimal_places` (int), `format` (custom Go time layout string) |

### Return Schema

```json
{
  "formatted": "1,234,567.89",
  "locale": "en-US",
  "kind": "number"
}
```

### Examples

```json
// Number with locale
{ "value": "1234567.89", "kind": "number", "locale": "de-DE" }
// → { "formatted": "1.234.567,89", "locale": "de-DE", "kind": "number" }

// Currency
{ "value": "1234.56", "kind": "currency", "locale": "en-US", "currency": "USD" }
// → { "formatted": "$1,234.56", "locale": "en-US", "kind": "currency" }

// Date
{ "value": "2024-03-15", "kind": "date", "locale": "de-DE" }
// → { "formatted": "15.03.2024", "locale": "de-DE", "kind": "date" }

// Percent
{ "value": "0.8542", "kind": "percent", "locale": "en-US" }
// → { "formatted": "85.4%", "locale": "en-US", "kind": "percent" }
```

---

## `frontend_icu_format`

Evaluate an ICU message format string with variable bindings. Supports simple substitution, plural rules, and select constructs.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `template` | string | ✓ | — | ICU message format template |
| `values` | object | ✓ | — | Variable bindings as key-value pairs |
| `locale` | string | | `en` | Locale for plural rules |

### Supported ICU Constructs

| Construct | Syntax | Example |
|-----------|--------|---------|
| Simple variable | `{varName}` | `Hello, {name}!` |
| Plural | `{var, plural, one{…} other{…}}` | `{n, plural, one{# item} other{# items}}` |
| Select | `{var, select, key{…} other{…}}` | `{gender, select, male{He} female{She} other{They}}` |

**`#` in plural clauses** is replaced by the numeric value.

**Plural categories** supported: `one`, `other` (English-like rules). Languages without plural distinction (`ja`, `zh`, `ko`, `vi`, `tr`, `id`) always use `other`.

### Return Schema

```json
{ "result": "You have 3 new messages" }
```

### Examples

```json
// Simple substitution
{
  "template": "Hello, {name}! You have {count} notifications.",
  "values": { "name": "Alice", "count": 7 }
}
// → { "result": "Hello, Alice! You have 7 notifications." }

// Plural
{
  "template": "You have {count, plural, one{# new message} other{# new messages}}.",
  "values": { "count": 1 },
  "locale": "en"
}
// → { "result": "You have 1 new message." }

// Select
{
  "template": "{gender, select, male{He submitted} female{She submitted} other{They submitted}} the form.",
  "values": { "gender": "female" }
}
// → { "result": "She submitted the form." }
```

---

## Error Responses

All tools return `{"error": "<message>"}` for invalid input. No panic, no stack trace.

| Tool | Common Error Triggers |
|------|-----------------------|
| `frontend_color` | Missing `color`, unsupported format, invalid `against` color |
| `frontend_css_unit` | Missing `from` or `to`, unknown unit name, zero viewport/parent for relative units |
| `frontend_breakpoint` | Negative `width`, unknown `system`, `custom` system with no `custom_breakpoints` |
| `frontend_regex` | Missing `pattern`, invalid RE2 syntax, unknown `operation` |
| `frontend_locale_format` | Missing `value` or `kind`, `currency` kind without `currency` code, non-numeric value for number/percent/currency |
| `frontend_icu_format` | Missing `template`, unmatched `{` in template, non-numeric plural variable |
