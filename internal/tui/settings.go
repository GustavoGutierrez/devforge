package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/config"
)

type settingsModel struct {
	cfg        *config.Config
	apiKey     string // masked input buffer
	imageModel string // plain text input buffer
	field      int
	status     string
	err        string
	goHome     bool
	saved      bool
}

func newSettingsModel(cfg *config.Config) settingsModel {
	key := ""
	imageModel := ""
	if cfg != nil {
		key = cfg.GeminiAPIKey
		imageModel = cfg.ImageModel
	}
	return settingsModel{cfg: cfg, apiKey: key, imageModel: imageModel}
}

func (m settingsModel) Init() tea.Cmd { return nil }

func (m settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case settingsSavedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.status = "Saved successfully."
			m.saved = true
			if m.cfg != nil {
				m.cfg.GeminiAPIKey = m.apiKey
				m.cfg.ImageModel = m.imageModel
			}
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.goHome = true
		case "tab":
			m.field = (m.field + 1) % 4
		case "shift+tab":
			m.field = (m.field + 3) % 4
		case "enter":
			switch m.field {
			case 2: // Save
				return m, m.save()
			case 3: // Delete key
				m.apiKey = ""
				return m, m.save()
			}
		case "backspace":
			if m.field == 0 && len(m.apiKey) > 0 {
				m.apiKey = m.apiKey[:len(m.apiKey)-1]
			}
			if m.field == 1 && len(m.imageModel) > 0 {
				m.imageModel = m.imageModel[:len(m.imageModel)-1]
			}
		default:
			if msg.Paste {
				if m.field == 0 {
					m.apiKey += string(msg.Runes)
				}
				if m.field == 1 {
					m.imageModel += string(msg.Runes)
				}
			} else {
				if m.field == 0 && len(msg.String()) == 1 {
					m.apiKey += msg.String()
				}
				if m.field == 1 && len(msg.String()) == 1 {
					m.imageModel += msg.String()
				}
			}
		}
	}
	return m, nil
}

type settingsSavedMsg struct{ err error }

func (m settingsModel) save() tea.Cmd {
	return func() tea.Msg {
		cfg := m.cfg
		if cfg == nil {
			cfg = &config.Config{
				ImageModel: "gemini-2.5-flash-image",
			}
		}
		cfg.GeminiAPIKey = m.apiKey
		cfg.ImageModel = m.imageModel
		err := config.Save(cfg)
		return settingsSavedMsg{err: err}
	}
}

func (m settingsModel) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Settings") + "\n\n")

	// Config path
	b.WriteString(dimStyle.Render("Config file: "+config.Path()) + "\n\n")

	// API key status
	statusIcon := "✗"
	statusColor := errorStyle
	if m.cfg != nil && m.cfg.GeminiAPIKey != "" {
		statusIcon = "✓"
		statusColor = successStyle
	}
	b.WriteString(fmt.Sprintf("Gemini API key: %s\n\n", statusColor.Render(statusIcon)))

	// Fields
	cursor0, cursor1, cursor2, cursor3 := "  ", "  ", "  ", "  "
	switch m.field {
	case 0:
		cursor0 = "> "
	case 1:
		cursor1 = "> "
	case 2:
		cursor2 = "> "
	case 3:
		cursor3 = "> "
	}

	// Masked key input
	maskedKey := strings.Repeat("*", len(m.apiKey))
	if m.field == 0 {
		maskedKey += "_"
		b.WriteString(selectedStyle.Render(cursor0+"API Key: "+maskedKey) + "\n")
	} else {
		b.WriteString(normalStyle.Render(cursor0+"API Key: "+maskedKey) + "\n")
	}

	// Image model input
	imageModelVal := m.imageModel
	if m.field == 1 {
		imageModelVal += "_"
		b.WriteString(selectedStyle.Render(cursor1+"Image Model: "+imageModelVal) + "\n")
	} else {
		b.WriteString(normalStyle.Render(cursor1+"Image Model: "+imageModelVal) + "\n")
	}

	// Save button
	if m.field == 2 {
		b.WriteString(selectedStyle.Render(cursor2+"[Save]") + "\n")
	} else {
		b.WriteString(normalStyle.Render(cursor2+"[Save]") + "\n")
	}

	// Delete button
	if m.field == 3 {
		b.WriteString(selectedStyle.Render(cursor3+"[Delete key]") + "\n")
	} else {
		b.WriteString(normalStyle.Render(cursor3+"[Delete key]") + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("Tab move • Enter activate • Esc back"))

	if m.status != "" {
		b.WriteString("\n\n" + successStyle.Render(m.status))
	}
	if m.err != "" {
		b.WriteString("\n\n" + errorStyle.Render(m.err))
	}

	return b.String()
}
