package tools

import (
	"context"
	"strings"
)

// AnalyzeLayoutInput is the input schema for the analyze_layout tool.
type AnalyzeLayoutInput struct {
	Markup      string    `json:"markup"`
	Stack       StackMeta `json:"stack"`
	PageType    string    `json:"page_type,omitempty"`
	DeviceFocus string    `json:"device_focus,omitempty"`
}

// StackMeta represents the CSS/framework stack metadata.
type StackMeta struct {
	CSSMode   string `json:"css_mode"`
	Framework string `json:"framework"`
}

// LayoutIssue represents a single issue found in the layout analysis.
type LayoutIssue struct {
	Severity    string `json:"severity"` // error | warning | suggestion
	Category    string `json:"category"` // spacing, typography, accessibility, etc.
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// AnalyzeLayoutOutput is the output schema for the analyze_layout tool.
type AnalyzeLayoutOutput struct {
	Summary string        `json:"summary"`
	Issues  []LayoutIssue `json:"issues"`
	Score   int           `json:"score"`
}

// AnalyzeLayout implements the analyze_layout MCP tool.
func (s *Server) AnalyzeLayout(ctx context.Context, input AnalyzeLayoutInput) string {
	if strings.TrimSpace(input.Markup) == "" {
		return errorJSON("markup is required")
	}
	if input.Stack.CSSMode == "" {
		return errorJSON("stack.css_mode is required")
	}
	if input.Stack.Framework == "" {
		return errorJSON("stack.framework is required")
	}

	issues := analyzeMarkup(input.Markup, input.Stack, input.PageType, input.DeviceFocus)
	score := calculateScore(issues)

	summary := buildSummary(issues, input.Stack)

	result := AnalyzeLayoutOutput{
		Summary: summary,
		Issues:  issues,
		Score:   score,
	}

	return mustJSON(result)
}

// analyzeMarkup inspects the markup string for common layout issues.
func analyzeMarkup(markup string, stack StackMeta, pageType, deviceFocus string) []LayoutIssue {
	var issues []LayoutIssue
	lower := strings.ToLower(markup)

	// Accessibility checks
	if !strings.Contains(lower, "alt=") && strings.Contains(lower, "<img") {
		issues = append(issues, LayoutIssue{
			Severity:    "error",
			Category:    "accessibility",
			Description: "One or more <img> elements are missing alt attributes.",
			Suggestion:  "Add descriptive alt text to all images, e.g. alt=\"Description of image\".",
		})
	}

	if !strings.Contains(lower, "aria-") && !strings.Contains(lower, "role=") {
		if strings.Contains(lower, "<nav") || strings.Contains(lower, "<header") || strings.Contains(lower, "<main") {
			// OK — semantic HTML is being used
		} else if strings.Contains(lower, "<div") {
			issues = append(issues, LayoutIssue{
				Severity:    "suggestion",
				Category:    "accessibility",
				Description: "Layout uses div elements without ARIA roles or semantic HTML tags.",
				Suggestion:  "Replace generic <div> containers with semantic elements like <nav>, <main>, <section>, <article>, or add appropriate role= attributes.",
			})
		}
	}

	// Typography checks
	if strings.Contains(lower, "font-size:") || strings.Contains(lower, "fontsize") {
		if !strings.Contains(lower, "rem") && !strings.Contains(lower, "em") {
			issues = append(issues, LayoutIssue{
				Severity:    "warning",
				Category:    "typography",
				Description: "Font size uses pixel units instead of relative units.",
				Suggestion:  "Use rem or em units for font sizes to respect user browser preferences.",
			})
		}
	}

	// Tailwind v4 specific checks
	if stack.CSSMode == "tailwind-v4" {
		if strings.Contains(markup, "tailwind.config") {
			issues = append(issues, LayoutIssue{
				Severity:    "error",
				Category:    "tailwind",
				Description: "Reference to tailwind.config.js detected in a Tailwind v4 project.",
				Suggestion:  "Tailwind v4 does not use tailwind.config.js. Move tokens to CSS @theme layer.",
			})
		}
		if strings.Contains(markup, "@apply") && !strings.Contains(markup, "@layer") {
			issues = append(issues, LayoutIssue{
				Severity:    "warning",
				Category:    "tailwind",
				Description: "@apply directive used outside of a @layer block.",
				Suggestion:  "In Tailwind v4, wrap @apply directives inside @layer components or @layer utilities.",
			})
		}
	}

	// Plain CSS specific checks
	if stack.CSSMode == "plain-css" {
		if strings.Contains(lower, "class=\"tw-") || strings.Contains(lower, "class=\"flex ") {
			issues = append(issues, LayoutIssue{
				Severity:    "warning",
				Category:    "consistency",
				Description: "Possible Tailwind utility classes detected in a plain-css project.",
				Suggestion:  "Use CSS custom properties (--color-primary, etc.) instead of Tailwind utility classes.",
			})
		}
	}

	// Spacing checks
	if strings.Contains(lower, "margin: 0px") || strings.Contains(lower, "padding: 0px") {
		issues = append(issues, LayoutIssue{
			Severity:    "suggestion",
			Category:    "spacing",
			Description: "Explicit 0px spacing found. Use 0 instead of 0px (shorthand without unit).",
			Suggestion:  "Replace 0px with 0 for zero-value spacing properties.",
		})
	}

	// Mobile responsiveness
	if deviceFocus == "mobile" || deviceFocus == "both" {
		if !strings.Contains(lower, "viewport") && !strings.Contains(lower, "meta") {
			if !strings.Contains(lower, "@media") && !strings.Contains(lower, "sm:") && !strings.Contains(lower, "md:") {
				issues = append(issues, LayoutIssue{
					Severity:    "warning",
					Category:    "responsive",
					Description: "No responsive breakpoints or media queries detected.",
					Suggestion:  "Add responsive classes (sm:, md:, lg: in Tailwind v4) or @media queries for mobile support.",
				})
			}
		}
	}

	// If no issues found, add a positive note
	if len(issues) == 0 {
		issues = []LayoutIssue{}
	}

	return issues
}

func calculateScore(issues []LayoutIssue) int {
	score := 100
	for _, issue := range issues {
		switch issue.Severity {
		case "error":
			score -= 20
		case "warning":
			score -= 10
		case "suggestion":
			score -= 3
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}

func buildSummary(issues []LayoutIssue, stack StackMeta) string {
	if len(issues) == 0 {
		return "No issues found. Layout looks good for " + stack.Framework + " with " + stack.CSSMode + "."
	}

	errors, warnings, suggestions := 0, 0, 0
	for _, issue := range issues {
		switch issue.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		case "suggestion":
			suggestions++
		}
	}

	parts := []string{}
	if errors > 0 {
		parts = append(parts, numStr(errors, "error"))
	}
	if warnings > 0 {
		parts = append(parts, numStr(warnings, "warning"))
	}
	if suggestions > 0 {
		parts = append(parts, numStr(suggestions, "suggestion"))
	}

	return "Found " + strings.Join(parts, ", ") + " in " + stack.Framework + "/" + stack.CSSMode + " layout."
}

func numStr(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strings.TrimPrefix(string(rune('0'+n)), "") + " " + word + "s"
}
