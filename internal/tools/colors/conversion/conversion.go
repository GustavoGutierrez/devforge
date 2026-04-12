// Package conversion provides standards-based color code conversion utilities.
//
// Conversion architecture follows a hub-and-spoke model:
//  1. Parse input into a canonical intermediate space (Linear sRGB).
//  2. Convert from Linear sRGB to the requested destination space.
//
// This keeps conversion logic consistent, extensible, and numerically stable.
package conversion

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ConvertInput is the MCP input schema for color code conversion.
type ConvertInput struct {
	Color string `json:"color"`
	From  string `json:"from"`
	To    string `json:"to"`
}

// ConvertOutput is the MCP output schema for color code conversion.
type ConvertOutput struct {
	Input      string             `json:"input"`
	From       string             `json:"from"`
	To         string             `json:"to"`
	Result     string             `json:"result"`
	Components map[string]float64 `json:"components,omitempty"`
}

// RGBFloat is gamma-encoded sRGB in range [0,1].
type RGBFloat struct{ R, G, B float64 }

// LinearRGB is linear-light sRGB in range [0,1] for in-gamut colors.
type LinearRGB struct{ R, G, B float64 }

// XYZ is CIE 1931 XYZ (D65), scaled as Y=100 for white.
type XYZ struct{ X, Y, Z float64 }

// LAB is CIE L*a*b* (D65 reference white).
type LAB struct{ L, A, B float64 }

// LCH is cylindrical LAB.
type LCH struct{ L, C, H float64 }

// OKLAB is Oklab perceptual color space.
type OKLAB struct{ L, A, B float64 }

// OKLCH is cylindrical Oklab.
type OKLCH struct{ L, C, H float64 }

// HSL is web HSL representation.
type HSL struct{ H, S, L float64 }

// HSV is web HSV representation.
type HSV struct{ H, S, V float64 }

// HWB is web HWB representation.
type HWB struct{ H, W, B float64 }

// Convert performs color conversion and returns JSON output.
func Convert(_ context.Context, in ConvertInput) string {
	out, err := Compute(in)
	if err != nil {
		return errJSON(err.Error())
	}
	return resultJSON(out)
}

// Compute performs strict color conversion via a Linear sRGB hub.
func Compute(in ConvertInput) (ConvertOutput, error) {
	if strings.TrimSpace(in.Color) == "" {
		return ConvertOutput{}, fmt.Errorf("color is required")
	}
	if strings.TrimSpace(in.From) == "" {
		return ConvertOutput{}, fmt.Errorf("from is required")
	}
	if strings.TrimSpace(in.To) == "" {
		return ConvertOutput{}, fmt.Errorf("to is required")
	}

	from, err := normalizeSpaceName(in.From)
	if err != nil {
		return ConvertOutput{}, err
	}
	to, err := normalizeSpaceName(in.To)
	if err != nil {
		return ConvertOutput{}, err
	}

	lin, err := parseToLinear(in.Color, from)
	if err != nil {
		return ConvertOutput{}, err
	}

	result, components, err := fromLinear(lin, to)
	if err != nil {
		return ConvertOutput{}, err
	}

	return ConvertOutput{
		Input:      in.Color,
		From:       from,
		To:         to,
		Result:     result,
		Components: components,
	}, nil
}

func normalizeSpaceName(name string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "-", "_")
	n = strings.ReplaceAll(n, " ", "_")

	aliases := map[string]string{
		"srgb":        "rgb",
		"linearrgb":   "linear_rgb",
		"linear_srgb": "linear_rgb",
		"linearsrgb":  "linear_rgb",
		"ok_lab":      "oklab",
		"ok_lch":      "oklch",
	}
	if canonical, ok := aliases[n]; ok {
		n = canonical
	}

	supported := map[string]bool{
		"hex":        true,
		"rgb":        true,
		"linear_rgb": true,
		"hsl":        true,
		"hsv":        true,
		"hwb":        true,
		"xyz":        true,
		"lab":        true,
		"lch":        true,
		"oklab":      true,
		"oklch":      true,
	}
	if !supported[n] {
		return "", fmt.Errorf("unsupported color space %q", name)
	}
	return n, nil
}

func parseToLinear(input, from string) (LinearRGB, error) {
	switch from {
	case "hex":
		r, g, b, err := parseHex(input)
		if err != nil {
			return LinearRGB{}, err
		}
		return RGBToLinear(RGBFloat{R: r / 255, G: g / 255, B: b / 255}), nil
	case "rgb":
		vals, err := parseTuple(input, "rgb", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		if !between(vals[0], 0, 255) || !between(vals[1], 0, 255) || !between(vals[2], 0, 255) {
			return LinearRGB{}, fmt.Errorf("rgb values must be in [0,255]")
		}
		return RGBToLinear(RGBFloat{R: vals[0] / 255, G: vals[1] / 255, B: vals[2] / 255}), nil
	case "linear_rgb":
		vals, err := parseTuple(input, "linear_rgb", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		if !between(vals[0], 0, 1) || !between(vals[1], 0, 1) || !between(vals[2], 0, 1) {
			return LinearRGB{}, fmt.Errorf("linear_rgb values must be in [0,1]")
		}
		return LinearRGB{R: vals[0], G: vals[1], B: vals[2]}, nil
	case "hsl":
		hsl, err := parseHSL(input)
		if err != nil {
			return LinearRGB{}, err
		}
		return RGBToLinear(HSLToRGB(hsl)), nil
	case "hsv":
		hsv, err := parseHSV(input)
		if err != nil {
			return LinearRGB{}, err
		}
		return RGBToLinear(HSVToRGB(hsv)), nil
	case "hwb":
		hwb, err := parseHWB(input)
		if err != nil {
			return LinearRGB{}, err
		}
		return RGBToLinear(HWBToRGB(hwb)), nil
	case "xyz":
		vals, err := parseTuple(input, "xyz", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		return XYZToLinear(XYZ{X: vals[0], Y: vals[1], Z: vals[2]}), nil
	case "lab":
		vals, err := parseTuple(input, "lab", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		return XYZToLinear(LABToXYZ(LAB{L: vals[0], A: vals[1], B: vals[2]})), nil
	case "lch":
		vals, err := parseTuple(input, "lch", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		return XYZToLinear(LABToXYZ(LCHToLAB(LCH{L: vals[0], C: vals[1], H: vals[2]}))), nil
	case "oklab":
		vals, err := parseTuple(input, "oklab", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		return OKLABToLinear(OKLAB{L: vals[0], A: vals[1], B: vals[2]}), nil
	case "oklch":
		vals, err := parseTuple(input, "oklch", 3)
		if err != nil {
			return LinearRGB{}, err
		}
		return OKLABToLinear(OKLCHToOKLAB(OKLCH{L: vals[0], C: vals[1], H: vals[2]})), nil
	default:
		return LinearRGB{}, fmt.Errorf("unsupported source color space %q", from)
	}
}

func fromLinear(lin LinearRGB, to string) (string, map[string]float64, error) {
	switch to {
	case "hex":
		srgb := LinearToRGB(lin)
		r, g, b := srgb255(srgb)
		return fmt.Sprintf("#%02X%02X%02X", r, g, b), map[string]float64{"r": float64(r), "g": float64(g), "b": float64(b)}, nil
	case "rgb":
		srgb := LinearToRGB(lin)
		r, g, b := srgb255(srgb)
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b), map[string]float64{"r": float64(r), "g": float64(g), "b": float64(b)}, nil
	case "linear_rgb":
		return fmt.Sprintf("linear_rgb(%s, %s, %s)", fmtFloat(lin.R), fmtFloat(lin.G), fmtFloat(lin.B)), map[string]float64{"r": lin.R, "g": lin.G, "b": lin.B}, nil
	case "hsl":
		hsl := RGBToHSL(LinearToRGBClamped(lin))
		return fmt.Sprintf("hsl(%s, %s%%, %s%%)", fmtFloat(hsl.H), fmtFloat(hsl.S*100), fmtFloat(hsl.L*100)), map[string]float64{"h": hsl.H, "s": hsl.S, "l": hsl.L}, nil
	case "hsv":
		hsv := RGBToHSV(LinearToRGBClamped(lin))
		return fmt.Sprintf("hsv(%s, %s%%, %s%%)", fmtFloat(hsv.H), fmtFloat(hsv.S*100), fmtFloat(hsv.V*100)), map[string]float64{"h": hsv.H, "s": hsv.S, "v": hsv.V}, nil
	case "hwb":
		hwb := RGBToHWB(LinearToRGBClamped(lin))
		return fmt.Sprintf("hwb(%s, %s%%, %s%%)", fmtFloat(hwb.H), fmtFloat(hwb.W*100), fmtFloat(hwb.B*100)), map[string]float64{"h": hwb.H, "w": hwb.W, "b": hwb.B}, nil
	case "xyz":
		xyz := LinearToXYZ(lin)
		return fmt.Sprintf("xyz(%s, %s, %s)", fmtFloat(xyz.X), fmtFloat(xyz.Y), fmtFloat(xyz.Z)), map[string]float64{"x": xyz.X, "y": xyz.Y, "z": xyz.Z}, nil
	case "lab":
		lab := XYZToLAB(LinearToXYZ(lin))
		return fmt.Sprintf("lab(%s, %s, %s)", fmtFloat(lab.L), fmtFloat(lab.A), fmtFloat(lab.B)), map[string]float64{"l": lab.L, "a": lab.A, "b": lab.B}, nil
	case "lch":
		lch := LABToLCH(XYZToLAB(LinearToXYZ(lin)))
		return fmt.Sprintf("lch(%s, %s, %s)", fmtFloat(lch.L), fmtFloat(lch.C), fmtFloat(lch.H)), map[string]float64{"l": lch.L, "c": lch.C, "h": lch.H}, nil
	case "oklab":
		ok := LinearToOKLAB(lin)
		return fmt.Sprintf("oklab(%s, %s, %s)", fmtFloat(ok.L), fmtFloat(ok.A), fmtFloat(ok.B)), map[string]float64{"l": ok.L, "a": ok.A, "b": ok.B}, nil
	case "oklch":
		oklch := OKLABToOKLCH(LinearToOKLAB(lin))
		return fmt.Sprintf("oklch(%s, %s, %s)", fmtFloat(oklch.L), fmtFloat(oklch.C), fmtFloat(oklch.H)), map[string]float64{"l": oklch.L, "c": oklch.C, "h": oklch.H}, nil
	default:
		return "", nil, fmt.Errorf("unsupported destination color space %q", to)
	}
}

// RGBToLinear converts gamma-encoded sRGB to linear sRGB.
func RGBToLinear(c RGBFloat) LinearRGB {
	return LinearRGB{
		R: inverseGamma(c.R),
		G: inverseGamma(c.G),
		B: inverseGamma(c.B),
	}
}

// LinearToRGB converts linear sRGB to gamma-encoded sRGB.
func LinearToRGB(c LinearRGB) RGBFloat {
	return RGBFloat{
		R: gammaEncode(c.R),
		G: gammaEncode(c.G),
		B: gammaEncode(c.B),
	}
}

// LinearToRGBClamped converts linear sRGB to gamma-encoded sRGB and clamps to [0,1].
func LinearToRGBClamped(c LinearRGB) RGBFloat {
	s := LinearToRGB(c)
	return RGBFloat{
		R: clamp(s.R, 0, 1),
		G: clamp(s.G, 0, 1),
		B: clamp(s.B, 0, 1),
	}
}

func inverseGamma(v float64) float64 {
	v = clamp(v, 0, 1)
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func gammaEncode(v float64) float64 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

// LinearToXYZ converts linear sRGB to CIE XYZ (D65, scaled by 100).
func LinearToXYZ(c LinearRGB) XYZ {
	return XYZ{
		X: (0.4124564*c.R + 0.3575761*c.G + 0.1804375*c.B) * 100.0,
		Y: (0.2126729*c.R + 0.7151522*c.G + 0.0721750*c.B) * 100.0,
		Z: (0.0193339*c.R + 0.1191920*c.G + 0.9503041*c.B) * 100.0,
	}
}

// XYZToLinear converts CIE XYZ (D65, scaled by 100) to linear sRGB.
func XYZToLinear(c XYZ) LinearRGB {
	x := c.X / 100.0
	y := c.Y / 100.0
	z := c.Z / 100.0
	return LinearRGB{
		R: 3.2404542*x + -1.5371385*y + -0.4985314*z,
		G: -0.9692660*x + 1.8760108*y + 0.0415560*z,
		B: 0.0556434*x + -0.2040259*y + 1.0572252*z,
	}
}

// XYZToLAB converts XYZ(D65) to CIE L*a*b*.
func XYZToLAB(c XYZ) LAB {
	const refX = 95.047
	const refY = 100.000
	const refZ = 108.883

	f := func(t float64) float64 {
		delta := 6.0 / 29.0
		if t > delta*delta*delta {
			return math.Cbrt(t)
		}
		return t/(3*delta*delta) + 4.0/29.0
	}

	x, y, z := f(c.X/refX), f(c.Y/refY), f(c.Z/refZ)
	return LAB{
		L: 116.0*y - 16.0,
		A: 500.0 * (x - y),
		B: 200.0 * (y - z),
	}
}

// LABToXYZ converts CIE L*a*b* to XYZ(D65).
func LABToXYZ(c LAB) XYZ {
	const refX = 95.047
	const refY = 100.000
	const refZ = 108.883

	inv := func(t float64) float64 {
		delta := 6.0 / 29.0
		if t > delta {
			return t * t * t
		}
		return 3 * delta * delta * (t - 4.0/29.0)
	}

	fy := (c.L + 16.0) / 116.0
	fx := fy + c.A/500.0
	fz := fy - c.B/200.0

	return XYZ{
		X: refX * inv(fx),
		Y: refY * inv(fy),
		Z: refZ * inv(fz),
	}
}

// LABToLCH converts LAB to cylindrical LCH.
func LABToLCH(c LAB) LCH {
	h := math.Atan2(c.B, c.A) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return LCH{L: c.L, C: math.Sqrt(c.A*c.A + c.B*c.B), H: h}
}

// LCHToLAB converts cylindrical LCH to LAB.
func LCHToLAB(c LCH) LAB {
	hRad := c.H * math.Pi / 180
	return LAB{
		L: c.L,
		A: c.C * math.Cos(hRad),
		B: c.C * math.Sin(hRad),
	}
}

// LinearToOKLAB converts linear sRGB to Oklab.
func LinearToOKLAB(c LinearRGB) OKLAB {
	l := 0.4122214708*c.R + 0.5363325363*c.G + 0.0514459929*c.B
	m := 0.2119034982*c.R + 0.6806995451*c.G + 0.1073969566*c.B
	s := 0.0883024619*c.R + 0.2817188376*c.G + 0.6299787005*c.B

	l3 := math.Cbrt(l)
	m3 := math.Cbrt(m)
	s3 := math.Cbrt(s)

	return OKLAB{
		L: 0.2104542553*l3 + 0.7936177850*m3 - 0.0040720468*s3,
		A: 1.9779984951*l3 - 2.4285922050*m3 + 0.4505937099*s3,
		B: 0.0259040371*l3 + 0.7827717662*m3 - 0.8086757660*s3,
	}
}

// OKLABToLinear converts Oklab to linear sRGB.
func OKLABToLinear(c OKLAB) LinearRGB {
	l := c.L + 0.3963377774*c.A + 0.2158037573*c.B
	m := c.L - 0.1055613458*c.A - 0.0638541728*c.B
	s := c.L - 0.0894841775*c.A - 1.2914855480*c.B

	l3 := l * l * l
	m3 := m * m * m
	s3 := s * s * s

	return LinearRGB{
		R: 4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3,
		G: -1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3,
		B: -0.0041960863*l3 - 0.7034186147*m3 + 1.7076147010*s3,
	}
}

// OKLABToOKLCH converts Oklab to cylindrical Oklch.
func OKLABToOKLCH(c OKLAB) OKLCH {
	h := math.Atan2(c.B, c.A) * 180 / math.Pi
	if h < 0 {
		h += 360
	}
	return OKLCH{L: c.L, C: math.Sqrt(c.A*c.A + c.B*c.B), H: h}
}

// OKLCHToOKLAB converts Oklch to Oklab.
func OKLCHToOKLAB(c OKLCH) OKLAB {
	hRad := c.H * math.Pi / 180
	return OKLAB{L: c.L, A: c.C * math.Cos(hRad), B: c.C * math.Sin(hRad)}
}

// RGBToHSL converts sRGB to HSL.
func RGBToHSL(c RGBFloat) HSL {
	maxV := math.Max(c.R, math.Max(c.G, c.B))
	minV := math.Min(c.R, math.Min(c.G, c.B))
	h, s, l := 0.0, 0.0, (maxV+minV)/2.0

	if maxV != minV {
		d := maxV - minV
		if l > 0.5 {
			s = d / (2.0 - maxV - minV)
		} else {
			s = d / (maxV + minV)
		}
		switch maxV {
		case c.R:
			h = (c.G - c.B) / d
			if c.G < c.B {
				h += 6
			}
		case c.G:
			h = (c.B-c.R)/d + 2
		case c.B:
			h = (c.R-c.G)/d + 4
		}
		h *= 60
	}

	return HSL{H: h, S: s, L: l}
}

// HSLToRGB converts HSL to sRGB.
func HSLToRGB(c HSL) RGBFloat {
	h := normalizeHue(c.H) / 360.0
	s := clamp(c.S, 0, 1)
	l := clamp(c.L, 0, 1)
	if s == 0 {
		return RGBFloat{R: l, G: l, B: l}
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	h2rgb := func(t float64) float64 {
		if t < 0 {
			t += 1
		}
		if t > 1 {
			t -= 1
		}
		switch {
		case t < 1.0/6.0:
			return p + (q-p)*6*t
		case t < 1.0/2.0:
			return q
		case t < 2.0/3.0:
			return p + (q-p)*(2.0/3.0-t)*6
		default:
			return p
		}
	}

	return RGBFloat{R: h2rgb(h + 1.0/3.0), G: h2rgb(h), B: h2rgb(h - 1.0/3.0)}
}

// RGBToHSV converts sRGB to HSV.
func RGBToHSV(c RGBFloat) HSV {
	maxV := math.Max(c.R, math.Max(c.G, c.B))
	minV := math.Min(c.R, math.Min(c.G, c.B))
	delta := maxV - minV

	h := 0.0
	if delta != 0 {
		switch maxV {
		case c.R:
			h = math.Mod((c.G-c.B)/delta, 6)
		case c.G:
			h = ((c.B-c.R)/delta + 2)
		case c.B:
			h = ((c.R-c.G)/delta + 4)
		}
		h *= 60
		if h < 0 {
			h += 360
		}
	}

	s := 0.0
	if maxV != 0 {
		s = delta / maxV
	}

	return HSV{H: h, S: s, V: maxV}
}

// HSVToRGB converts HSV to sRGB.
func HSVToRGB(c HSV) RGBFloat {
	h := normalizeHue(c.H)
	s := clamp(c.S, 0, 1)
	v := clamp(c.V, 0, 1)

	chroma := v * s
	x := chroma * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - chroma

	var r1, g1, b1 float64
	switch {
	case h >= 0 && h < 60:
		r1, g1, b1 = chroma, x, 0
	case h >= 60 && h < 120:
		r1, g1, b1 = x, chroma, 0
	case h >= 120 && h < 180:
		r1, g1, b1 = 0, chroma, x
	case h >= 180 && h < 240:
		r1, g1, b1 = 0, x, chroma
	case h >= 240 && h < 300:
		r1, g1, b1 = x, 0, chroma
	default:
		r1, g1, b1 = chroma, 0, x
	}

	return RGBFloat{R: r1 + m, G: g1 + m, B: b1 + m}
}

// RGBToHWB converts sRGB to HWB.
func RGBToHWB(c RGBFloat) HWB {
	h := RGBToHSV(c).H
	w := math.Min(c.R, math.Min(c.G, c.B))
	b := 1.0 - math.Max(c.R, math.Max(c.G, c.B))
	return HWB{H: h, W: w, B: b}
}

// HWBToRGB converts HWB to sRGB according to CSS Color spec behavior.
func HWBToRGB(c HWB) RGBFloat {
	h := normalizeHue(c.H)
	w := clamp(c.W, 0, 1)
	b := clamp(c.B, 0, 1)

	if w+b >= 1 {
		gray := w / (w + b)
		return RGBFloat{R: gray, G: gray, B: gray}
	}

	base := HSVToRGB(HSV{H: h, S: 1, V: 1})
	scale := 1 - w - b
	return RGBFloat{
		R: base.R*scale + w,
		G: base.G*scale + w,
		B: base.B*scale + w,
	}
}

func parseHex(input string) (r, g, b float64, err error) {
	s := strings.TrimSpace(strings.TrimPrefix(input, "#"))
	if len(s) == 3 {
		s = strings.Repeat(string(s[0]), 2) + strings.Repeat(string(s[1]), 2) + strings.Repeat(string(s[2]), 2)
	}
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("hex color must be #RGB or #RRGGBB")
	}
	v, e := strconv.ParseUint(s, 16, 32)
	if e != nil {
		return 0, 0, 0, fmt.Errorf("invalid hex color")
	}
	return float64(uint8(v >> 16)), float64(uint8((v >> 8) & 0xFF)), float64(uint8(v & 0xFF)), nil
}

func parseTuple(input, prefix string, expected int) ([]float64, error) {
	_ = prefix // kept for readability in call sites
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}

	if open := strings.Index(raw, "("); open >= 0 && strings.HasSuffix(raw, ")") {
		raw = raw[open+1 : len(raw)-1]
	}

	parts := strings.Split(raw, ",")
	if len(parts) != expected {
		return nil, fmt.Errorf("expected %d components", expected)
	}

	out := make([]float64, expected)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric component %q", strings.TrimSpace(p))
		}
		out[i] = v
	}
	return out, nil
}

func parseHSL(input string) (HSL, error) {
	vals, err := parseTupleWithPercent(input, "hsl", 3, map[int]bool{1: true, 2: true})
	if err != nil {
		return HSL{}, err
	}
	return HSL{H: normalizeHue(vals[0]), S: normalizePercent(vals[1]), L: normalizePercent(vals[2])}, nil
}

func parseHSV(input string) (HSV, error) {
	vals, err := parseTupleWithPercent(input, "hsv", 3, map[int]bool{1: true, 2: true})
	if err != nil {
		return HSV{}, err
	}
	return HSV{H: normalizeHue(vals[0]), S: normalizePercent(vals[1]), V: normalizePercent(vals[2])}, nil
}

func parseHWB(input string) (HWB, error) {
	vals, err := parseTupleWithPercent(input, "hwb", 3, map[int]bool{1: true, 2: true})
	if err != nil {
		return HWB{}, err
	}
	w := normalizePercent(vals[1])
	b := normalizePercent(vals[2])
	if w+b > 1.0 {
		sum := w + b
		w /= sum
		b /= sum
	}
	return HWB{H: normalizeHue(vals[0]), W: w, B: b}, nil
}

func parseTupleWithPercent(input, prefix string, expected int, percentPositions map[int]bool) ([]float64, error) {
	_ = prefix // kept for readability in call sites
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}

	if open := strings.Index(raw, "("); open >= 0 && strings.HasSuffix(raw, ")") {
		raw = raw[open+1 : len(raw)-1]
	}

	parts := strings.Split(raw, ",")
	if len(parts) != expected {
		return nil, fmt.Errorf("expected %d components", expected)
	}

	out := make([]float64, expected)
	for i, p := range parts {
		token := strings.TrimSpace(p)
		if percentPositions[i] {
			token = strings.TrimSuffix(token, "%")
		}
		v, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric component %q", strings.TrimSpace(p))
		}
		out[i] = v
	}

	return out, nil
}

func normalizePercent(v float64) float64 {
	if v <= 1.0 {
		return clamp(v, 0, 1)
	}
	return clamp(v/100.0, 0, 1)
}

func normalizeHue(v float64) float64 {
	h := math.Mod(v, 360)
	if h < 0 {
		h += 360
	}
	return h
}

func srgb255(c RGBFloat) (int, int, int) {
	r := int(math.Round(clamp(c.R, 0, 1) * 255))
	g := int(math.Round(clamp(c.G, 0, 1) * 255))
	b := int(math.Round(clamp(c.B, 0, 1) * 255))
	return r, g, b
}

func fmtFloat(v float64) string {
	if math.Abs(v) < 1e-12 {
		v = 0
	}
	return strconv.FormatFloat(v, 'f', 6, 64)
}

func between(v, lo, hi float64) bool {
	return v >= lo && v <= hi
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func errJSON(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

func resultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return errJSON("marshal failed: " + err.Error())
	}
	return string(b)
}
