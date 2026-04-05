package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ManageTokensInput is the input schema for the manage_tokens tool.
type ManageTokensInput struct {
	Mode     string                 `json:"mode"`     // read | plan-update | apply-update
	CSSMode  string                 `json:"css_mode"` // tailwind-v4 | plain-css
	Scope    string                 `json:"scope"`    // colors | spacing | typography | all
	Proposal map[string]interface{} `json:"proposal,omitempty"`
}

// ManageTokensOutput is the output schema for the manage_tokens tool.
type ManageTokensOutput struct {
	CurrentTokens map[string]string            `json:"current_tokens"`
	Diff          map[string]map[string]string `json:"diff"`
	Instructions  string                       `json:"instructions"`
}

// ManageTokens implements the manage_tokens MCP tool.
func (s *Server) ManageTokens(ctx context.Context, input ManageTokensInput) string {
	if input.Mode == "" {
		return errorJSON("mode is required (read | plan-update | apply-update)")
	}
	if input.CSSMode == "" {
		return errorJSON("css_mode is required")
	}
	if input.Scope == "" {
		return errorJSON("scope is required")
	}

	switch input.Mode {
	case "read":
		return s.readTokens(ctx, input)
	case "plan-update":
		return s.planTokenUpdate(ctx, input)
	case "apply-update":
		return s.applyTokenUpdate(ctx, input)
	default:
		return errorJSON("invalid mode: " + input.Mode + " (use read | plan-update | apply-update)")
	}
}

func (s *Server) readTokens(ctx context.Context, input ManageTokensInput) string {
	currentTokens := defaultTokens(input.Scope)

	instructions := buildReadInstructions(input.CSSMode, currentTokens)

	return mustJSON(ManageTokensOutput{
		CurrentTokens: currentTokens,
		Diff:          map[string]map[string]string{},
		Instructions:  instructions,
	})
}

func (s *Server) planTokenUpdate(ctx context.Context, input ManageTokensInput) string {
	if len(input.Proposal) == 0 {
		return errorJSON("proposal is required for plan-update mode")
	}

	currentTokens := defaultTokens(input.Scope)

	diff := buildDiff(currentTokens, input.Proposal)
	instructions := buildUpdateInstructions(input.CSSMode, input.Proposal)

	return mustJSON(ManageTokensOutput{
		CurrentTokens: currentTokens,
		Diff:          diff,
		Instructions:  instructions,
	})
}

func (s *Server) applyTokenUpdate(ctx context.Context, input ManageTokensInput) string {
	if len(input.Proposal) == 0 {
		return errorJSON("proposal is required for apply-update mode")
	}

	currentTokens := defaultTokens(input.Scope)

	diff := buildDiff(currentTokens, input.Proposal)

	// Merge proposal into currentTokens for output
	for k, v := range input.Proposal {
		currentTokens[k] = fmt.Sprintf("%v", v)
	}

	instructions := buildUpdateInstructions(input.CSSMode, input.Proposal)

	return mustJSON(ManageTokensOutput{
		CurrentTokens: currentTokens,
		Diff:          diff,
		Instructions:  instructions,
	})
}

func defaultTokens(scope string) map[string]string {
	all := map[string]string{
		"--color-primary":    "#3b82f6",
		"--color-secondary":  "#8b5cf6",
		"--color-background": "#ffffff",
		"--color-surface":    "#f8fafc",
		"--color-text":       "#1e293b",
		"--color-muted":      "#64748b",
		"--spacing-xs":       "0.25rem",
		"--spacing-sm":       "0.5rem",
		"--spacing-md":       "1rem",
		"--spacing-lg":       "2rem",
		"--spacing-xl":       "4rem",
		"--font-size-sm":     "0.875rem",
		"--font-size-base":   "1rem",
		"--font-size-lg":     "1.125rem",
		"--font-size-xl":     "1.25rem",
		"--font-size-2xl":    "1.5rem",
		"--font-size-4xl":    "2.25rem",
	}

	if scope == "all" {
		return all
	}

	filtered := make(map[string]string)
	for k, v := range all {
		if strings.Contains(k, scope) || (scope == "colors" && strings.Contains(k, "--color")) ||
			(scope == "spacing" && strings.Contains(k, "--spacing")) ||
			(scope == "typography" && strings.Contains(k, "--font")) {
			filtered[k] = v
		}
	}
	return filtered
}

func buildDiff(current map[string]string, proposal map[string]interface{}) map[string]map[string]string {
	diff := make(map[string]map[string]string)
	for k, newVal := range proposal {
		newStr := fmt.Sprintf("%v", newVal)
		oldStr := current[k]
		if oldStr != newStr {
			diff[k] = map[string]string{
				"old": oldStr,
				"new": newStr,
			}
		}
	}
	return diff
}

func buildReadInstructions(cssMode string, tokens map[string]string) string {
	if cssMode == "tailwind-v4" {
		return "Apply tokens in @theme { } block in your global CSS file:\n\n@theme {\n  " +
			formatTokensCSS(tokens) + "\n}"
	}
	return "Apply tokens in :root { } block in your global CSS file:\n\n:root {\n  " +
		formatTokensCSS(tokens) + "\n}"
}

func buildUpdateInstructions(cssMode string, proposal map[string]interface{}) string {
	b, _ := json.Marshal(proposal)
	if cssMode == "tailwind-v4" {
		return "Add or update in @theme { } block:\n" + string(b)
	}
	return "Add or update in :root { } block:\n" + string(b)
}

func formatTokensCSS(tokens map[string]string) string {
	var parts []string
	for k, v := range tokens {
		parts = append(parts, fmt.Sprintf("%s: %s;", k, v))
	}
	return strings.Join(parts, "\n  ")
}
