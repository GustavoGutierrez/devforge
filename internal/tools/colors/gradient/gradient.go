// Package gradient provides CSS gradient generation utilities.
package gradient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ColorStopInput defines a single gradient stop.
// Position is optional and expressed in percentage [0,100].
type ColorStopInput struct {
	Color    string `json:"color"`
	Position *int   `json:"position,omitempty"`
}

// GenerateInput defines input parameters for CSS gradient generation.
type GenerateInput struct {
	GradientType string           `json:"gradient_type"` // linear | radial
	Angle        *int             `json:"angle,omitempty"`
	Shape        string           `json:"shape,omitempty"` // radial only: circle | ellipse
	Stops        []ColorStopInput `json:"stops"`
}

// GenerateOutput is the JSON response payload.
type GenerateOutput struct {
	GradientType string           `json:"gradient_type"`
	Angle        int              `json:"angle,omitempty"`
	Shape        string           `json:"shape,omitempty"`
	Stops        []ColorStopInput `json:"stops"`
	Fallback     string           `json:"fallback"`
	Gradient     string           `json:"gradient"`
	CSS          string           `json:"css"`
}

// Generate builds a CSS gradient declaration and returns JSON output.
func Generate(_ context.Context, in GenerateInput) string {
	out, err := Compute(in)
	if err != nil {
		return errJSON(err.Error())
	}
	return resultJSON(out)
}

// Compute validates and builds a CSS gradient result.
func Compute(in GenerateInput) (GenerateOutput, error) {
	gt := strings.ToLower(strings.TrimSpace(in.GradientType))
	if gt == "" {
		return GenerateOutput{}, fmt.Errorf("gradient_type is required")
	}
	if gt != "linear" && gt != "radial" {
		return GenerateOutput{}, fmt.Errorf("gradient_type must be 'linear' or 'radial'")
	}

	if len(in.Stops) < 2 {
		return GenerateOutput{}, fmt.Errorf("at least 2 color stops are required")
	}

	normalizedStops := make([]ColorStopInput, 0, len(in.Stops))
	for i, s := range in.Stops {
		color := strings.TrimSpace(s.Color)
		if color == "" {
			return GenerateOutput{}, fmt.Errorf("stops[%d].color is required", i)
		}
		ns := ColorStopInput{Color: color}
		if s.Position != nil {
			p := clamp(*s.Position, 0, 100)
			ns.Position = &p
		}
		normalizedStops = append(normalizedStops, ns)
	}

	stopsCSS := make([]string, 0, len(normalizedStops))
	for _, s := range normalizedStops {
		if s.Position != nil {
			stopsCSS = append(stopsCSS, fmt.Sprintf("%s %d%%", s.Color, *s.Position))
		} else {
			stopsCSS = append(stopsCSS, s.Color)
		}
	}

	fallback := normalizedStops[0].Color
	joinedStops := strings.Join(stopsCSS, ", ")

	out := GenerateOutput{
		GradientType: gt,
		Stops:        normalizedStops,
		Fallback:     fallback,
	}

	if gt == "linear" {
		angle := 0
		if in.Angle != nil {
			angle = *in.Angle
		}
		out.Angle = angle
		out.Gradient = fmt.Sprintf("linear-gradient(%ddeg, %s)", angle, joinedStops)
	} else {
		shape := strings.ToLower(strings.TrimSpace(in.Shape))
		if shape == "" {
			shape = "circle"
		}
		if shape != "circle" && shape != "ellipse" {
			return GenerateOutput{}, fmt.Errorf("shape must be 'circle' or 'ellipse'")
		}
		out.Shape = shape
		out.Gradient = fmt.Sprintf("radial-gradient(%s, %s)", shape, joinedStops)
	}

	out.CSS = fmt.Sprintf("background: %s;\nbackground: %s;", fallback, out.Gradient)
	return out, nil
}

func clamp(v, lo, hi int) int {
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
