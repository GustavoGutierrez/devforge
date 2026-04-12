# Color Utilities — Tool Reference

> Group · Packages `dev-forge-mcp/internal/tools/colors/conversion`, `dev-forge-mcp/internal/tools/colors/harmony`, and `dev-forge-mcp/internal/tools/colors/gradient`

Stateless color utilities for standards-based color conversion, harmony palette generation, and CSS gradient generation.

---

## `color_code_convert`

Convert a color code between supported spaces using a hub-and-spoke conversion pipeline:

- Source input is parsed into **Linear sRGB** (or converted through **XYZ** when needed).
- Destination output is generated from that canonical intermediate representation.

This architecture avoids pairwise conversion drift and keeps results stable.

### Supported spaces

- `hex`
- `rgb`
- `linear_rgb`
- `hsl`
- `hsv`
- `hwb`
- `xyz`
- `lab`
- `lch`
- `oklab`
- `oklch`

### Parameters

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `color` | string | ✓ | Input color code string (for example: `#3B82F6`, `rgb(59,130,246)`, `lab(53.2,80.1,67.2)`) |
| `from` | string | ✓ | Source color space |
| `to` | string | ✓ | Destination color space |

### Return Schema

```json
{
  "input": "#3B82F6",
  "from": "hex",
  "to": "oklch",
  "result": "oklch(0.623097, 0.188014, 259.814526)",
  "components": {
    "l": 0.623097,
    "c": 0.188014,
    "h": 259.814526
  }
}
```

### Notes

- `hsl`, `hsv`, and `hwb` accept percentage syntax (for example `hsl(210, 65%, 40%)`).
- `hex` accepts short (`#RGB`) and full (`#RRGGBB`) notation.
- Aliases supported for linear space: `linear-srgb`, `linear_srgb`, `linearrgb`.

### Error Cases

- Missing `color`, `from`, or `to`
- Invalid input syntax for the selected source space
- Unsupported source or destination space

---

## `css_gradient_generate`

Generate CSS linear or radial gradients from two or more color stops.

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `gradient_type` | string | ✓ | — | `linear` or `radial` |
| `stops` | array | ✓ | — | Stop list with `{ color: string, position?: number }` |
| `angle` | number | | `0` | Linear angle in degrees |
| `shape` | string | | `circle` | Radial shape: `circle` or `ellipse` |

### Return Schema

```json
{
  "gradient_type": "linear",
  "angle": 90,
  "stops": [
    { "color": "#2a7b9b", "position": 0 },
    { "color": "#57c785", "position": 50 },
    { "color": "#eddd53", "position": 100 }
  ],
  "fallback": "#2a7b9b",
  "gradient": "linear-gradient(90deg, #2a7b9b 0%, #57c785 50%, #eddd53 100%)",
  "css": "background: #2a7b9b;\nbackground: linear-gradient(90deg, #2a7b9b 0%, #57c785 50%, #eddd53 100%);"
}
```

### Notes

- Requires at least 2 stops.
- Stop positions are optional; when omitted, CSS distributes them automatically.
- Stop positions are clamped to `[0,100]`.

### Error Cases

- Missing or invalid `gradient_type`
- Fewer than 2 stops
- Empty stop color
- Invalid radial `shape`

---

## `color_harmony_palette`

Generate a harmony-based 5-color palette from a base color.

Supported harmony types:

- `analogous`
- `monochromatic`
- `triad`
- `complementary`
- `split_complementary`
- `square`
- `compound`
- `shades`

### Parameters

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `base_color` | string | ✓ | — | Base HEX color (`#RRGGBB` or `#RGB`) |
| `harmony` | string | ✓ | — | Harmony model name |
| `spread` | number | | harmony-specific | Optional angle in degrees |

Default `spread` values (when omitted):

- `analogous`: `30`
- `triad`: `120`
- `split_complementary`: `150`
- `square`: `90`
- `compound`: `30`
- others: `0`

### Return Schema

```json
{
  "base_color": "#FF6B6B",
  "harmony": "analogous",
  "spread": 30,
  "colors": ["#RRGGBB", "#RRGGBB", "#RRGGBB", "#RRGGBB", "#RRGGBB"]
}
```

### Examples

```json
{
  "base_color": "#FF6B6B",
  "harmony": "analogous"
}
```

```json
{
  "base_color": "#3366CC",
  "harmony": "split_complementary",
  "spread": 160
}
```

### Error Cases

- Missing `base_color` or `harmony`
- Invalid HEX format
- Unsupported harmony type
- Negative `spread`
