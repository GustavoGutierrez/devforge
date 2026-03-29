package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// mcpSetupModel handles the MCP server auto-configuration flow.
type mcpSetupModel struct {
	step      int // 0=ide, 1=scope, 2=path, 3=done
	ideIdx    int // 0=OpenCode, 1=ClaudeCode, 2=VSCode
	scopeIdx  int // 0=global, 1=project-local
	pathInput string
	cursor    int
	result    string
	isError   bool
	goHome    bool
}

var ideNames = []string{"OpenCode", "Claude Code", "VSCode"}

func newMCPSetupModel() mcpSetupModel {
	return mcpSetupModel{
		step:      0,
		ideIdx:    0,
		scopeIdx:  0,
		pathInput: detectMCPBinaryPath(),
		cursor:    0,
	}
}

// detectMCPBinaryPath tries to find the dev-forge-mcp binary next to the current executable.
func detectMCPBinaryPath() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "dev-forge-mcp")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".local", "bin", "dev-forge-mcp")
	}
	return "/usr/local/bin/dev-forge-mcp"
}

func (m mcpSetupModel) Init() tea.Cmd { return nil }

func (m mcpSetupModel) Update(msg tea.Msg) (mcpSetupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case 0: // IDE selection
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(ideNames)-1 {
					m.cursor++
				}
			case "enter", " ":
				m.ideIdx = m.cursor
				m.cursor = 0
				if m.ideIdx == 2 { // VSCode — skip scope step
					m.step = 2
				} else {
					m.step = 1
				}
			case "esc", "q":
				m.goHome = true
			}

		case 1: // Scope selection
			scopes := scopesForIDE(m.ideIdx)
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(scopes)-1 {
					m.cursor++
				}
			case "enter", " ":
				m.scopeIdx = m.cursor
				m.cursor = 0
				m.step = 2
			case "esc":
				m.step = 0
				m.cursor = m.ideIdx
			}

		case 2: // Path confirmation
			switch msg.String() {
			case "enter":
				m.step = 3
				m.result, m.isError = writeConfig(m.ideIdx, m.scopeIdx, m.pathInput)
			case "backspace":
				if len(m.pathInput) > 0 {
					m.pathInput = m.pathInput[:len(m.pathInput)-1]
				}
			case "esc":
				if m.ideIdx == 2 { // VSCode skipped scope
					m.step = 0
					m.cursor = m.ideIdx
				} else {
					m.step = 1
					m.cursor = m.scopeIdx
				}
			default:
				if len(msg.String()) == 1 {
					m.pathInput += msg.String()
				}
			}

		case 3: // Done
			switch msg.String() {
			case "q", "esc", "enter", " ":
				m.goHome = true
			}
		}
	}
	return m, nil
}

func (m mcpSetupModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("◆ Configure MCP Server") + "\n\n")

	switch m.step {
	case 0:
		b.WriteString(normalStyle.Render("Select IDE:") + "\n")
		for i, name := range ideNames {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			if m.cursor == i {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("%s%s", cursor, name)) + "\n")
			} else {
				b.WriteString(normalStyle.Render(fmt.Sprintf("%s%s", cursor, name)) + "\n")
			}
		}
		b.WriteString(helpStyle.Render("\n↑/↓ navigate • Enter select • Esc/q back"))

	case 1:
		scopes := scopesForIDE(m.ideIdx)
		b.WriteString(dimStyle.Render(fmt.Sprintf("IDE: %s", ideNames[m.ideIdx])) + "\n\n")
		b.WriteString(normalStyle.Render("Select scope:") + "\n")
		for i, s := range scopes {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			label := fmt.Sprintf("%s%s", cursor, s.label)
			if m.cursor == i {
				b.WriteString(selectedStyle.Render(label) + "\n")
				b.WriteString(dimStyle.Render("    "+s.path) + "\n")
			} else {
				b.WriteString(normalStyle.Render(label) + "\n")
				b.WriteString(dimStyle.Render("    "+s.path) + "\n")
			}
		}
		b.WriteString(helpStyle.Render("\n↑/↓ navigate • Enter select • Esc back"))

	case 2:
		b.WriteString(dimStyle.Render(fmt.Sprintf("IDE: %s  •  Scope: %s", ideNames[m.ideIdx], scopeLabel(m.ideIdx, m.scopeIdx))) + "\n\n")
		b.WriteString(normalStyle.Render("Binary path (edit if needed):") + "\n")
		b.WriteString(selectedStyle.Render(m.pathInput+"_") + "\n")
		b.WriteString(helpStyle.Render("\nType to edit • Enter confirm • Esc back"))

	case 3:
		if m.isError {
			b.WriteString(errorStyle.Render("Error: "+m.result) + "\n")
		} else {
			b.WriteString(successStyle.Render("✓ "+m.result) + "\n")
		}
		b.WriteString(helpStyle.Render("\nPress any key to go back"))
	}

	return b.String()
}

// scope describes a config file target.
type scope struct {
	label string
	path  string
}

func scopesForIDE(ideIdx int) []scope {
	home, _ := os.UserHomeDir()
	switch ideIdx {
	case 0: // OpenCode
		return []scope{
			{label: "Global", path: filepath.Join(home, ".config", "opencode", "config.json")},
			{label: "Project-local", path: "./opencode.json"},
		}
	case 1: // Claude Code
		return []scope{
			{label: "Global", path: filepath.Join(home, ".claude.json")},
			{label: "Project-local", path: "./.mcp.json"},
		}
	}
	return []scope{}
}

func scopeLabel(ideIdx, scopeIdx int) string {
	if ideIdx == 2 {
		return "Project-local"
	}
	scopes := scopesForIDE(ideIdx)
	if scopeIdx < len(scopes) {
		return scopes[scopeIdx].label
	}
	return ""
}

func configFilePath(ideIdx, scopeIdx int) string {
	home, _ := os.UserHomeDir()
	switch ideIdx {
	case 0: // OpenCode
		if scopeIdx == 0 {
			return filepath.Join(home, ".config", "opencode", "config.json")
		}
		return "opencode.json"
	case 1: // Claude Code
		if scopeIdx == 0 {
			return filepath.Join(home, ".claude.json")
		}
		return ".mcp.json"
	case 2: // VSCode
		return filepath.Join(".vscode", "mcp.json")
	}
	return ""
}

// writeConfig merges the dev-forge MCP entry into the target config file.
// Returns (message, isError).
func writeConfig(ideIdx, scopeIdx int, binaryPath string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Sprintf("could not determine home directory: %v", err), true
	}
	devForgeConfigPath := filepath.Join(home, ".config", "devforge", "config.json")

	cfgPath := configFilePath(ideIdx, scopeIdx)

	// Read existing file or start fresh.
	var root map[string]interface{}
	data, err := os.ReadFile(cfgPath)
	if err == nil {
		if jsonErr := json.Unmarshal(data, &root); jsonErr != nil {
			return fmt.Sprintf("failed to parse existing %s: %v", cfgPath, jsonErr), true
		}
	}
	if root == nil {
		root = make(map[string]interface{})
	}

	switch ideIdx {
	case 0: // OpenCode
		entry := map[string]interface{}{
			"type":    "local",
			"command": []interface{}{binaryPath},
			"environment": map[string]interface{}{
				"DEV_FORGE_CONFIG": devForgeConfigPath,
			},
		}
		setNestedKey(root, entry, "mcp", "devforge")

	case 1: // Claude Code
		entry := map[string]interface{}{
			"command": binaryPath,
			"args":    []interface{}{},
			"env": map[string]interface{}{
				"DEV_FORGE_CONFIG": devForgeConfigPath,
			},
		}
		setNestedKey(root, entry, "mcpServers", "devforge")

	case 2: // VSCode
		entry := map[string]interface{}{
			"type":    "stdio",
			"command": binaryPath,
			"args":    []interface{}{},
			"env": map[string]interface{}{
				"DEV_FORGE_CONFIG": devForgeConfigPath,
			},
		}
		setNestedKey(root, entry, "servers", "devforge")
	}

	// Ensure parent dirs exist.
	if mkErr := os.MkdirAll(filepath.Dir(cfgPath), 0o755); mkErr != nil {
		return fmt.Sprintf("failed to create directory: %v", mkErr), true
	}

	out, marshalErr := json.MarshalIndent(root, "", "  ")
	if marshalErr != nil {
		return fmt.Sprintf("failed to marshal JSON: %v", marshalErr), true
	}

	if writeErr := os.WriteFile(cfgPath, append(out, '\n'), 0o644); writeErr != nil {
		return fmt.Sprintf("failed to write %s: %v", cfgPath, writeErr), true
	}

	return fmt.Sprintf("Written to %s", cfgPath), false
}

// setNestedKey sets root[topKey][entryKey] = value, creating intermediate maps as needed.
func setNestedKey(root map[string]interface{}, value interface{}, topKey, entryKey string) {
	top, ok := root[topKey]
	if !ok {
		top = make(map[string]interface{})
	}
	topMap, ok := top.(map[string]interface{})
	if !ok {
		topMap = make(map[string]interface{})
	}
	topMap[entryKey] = value
	root[topKey] = topMap
}
