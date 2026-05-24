// Package toolsutil provides shared helpers for all MCP tool handler packages.
package toolsutil

import (
	"encoding/json"

	"dev-forge-mcp/internal/dpf"
)

// ErrResult returns the canonical JSON error envelope {"error": "..."}.
func ErrResult(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}

// ResultJSON marshals v to a JSON string.
// On marshal failure it returns ErrResult with the marshal error.
func ResultJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ErrResult("marshal failed: " + err.Error())
	}
	return string(b)
}

// RequireDPF checks whether the dpf Streamer is ready.
// Returns ("", true) when client is non-nil; returns (errEnvelope, false) when nil.
func RequireDPF(client dpf.Streamer) (string, bool) {
	if client == nil {
		return ErrResult("dpf binary not available. Ensure bin/dpf is installed and executable."), false
	}
	return "", true
}
