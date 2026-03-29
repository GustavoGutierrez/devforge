package tui

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"dev-forge-mcp/internal/tools"
	"dev-forge-mcp/internal/tools/cryptoutil"
)

var cryptoOps = []string{
	"Hash (SHA-256/512/MD5/SHA-1)",
	"HMAC",
	"JWT (decode/verify/generate)",
	"Password Hash/Verify",
	"Key Generation (RSA/EC/Ed25519)",
	"Random Token/Bytes/OTP",
	"Mask Secrets",
}

type cryptoutilModel struct {
	phase    int
	opIdx    int
	fieldIdx int
	fields   []fieldDef
	values   map[string]string
	inputBuf string
	result   string
	isError  bool
	goHome   bool
	srv      *tools.Server
}

func newCryptoutilModel(srv *tools.Server) cryptoutilModel {
	return cryptoutilModel{srv: srv, values: make(map[string]string)}
}

func (m cryptoutilModel) Init() tea.Cmd { return nil }

func (m cryptoutilModel) Update(msg tea.Msg) (cryptoutilModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.phase {
		case 0:
			return m.updatePhase0(msg)
		case 1:
			return m.updatePhase1(msg)
		case 2:
			return m.updatePhase2(msg)
		}
	}
	return m, nil
}

func (m cryptoutilModel) updatePhase0(msg tea.KeyMsg) (cryptoutilModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.opIdx > 0 {
			m.opIdx--
		}
	case "down", "j":
		if m.opIdx < len(cryptoOps)-1 {
			m.opIdx++
		}
	case "enter", " ":
		m.fields = cryptoFields(m.opIdx)
		m.fieldIdx = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
		m.phase = 1
	case "esc", "q":
		m.goHome = true
	}
	return m, nil
}

func (m cryptoutilModel) updatePhase1(msg tea.KeyMsg) (cryptoutilModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.fieldIdx == 0 {
			m.phase = 0
			m.inputBuf = ""
		} else {
			m.fieldIdx--
			m.inputBuf = m.values[m.fields[m.fieldIdx].key]
		}
	case "enter":
		m.values[m.fields[m.fieldIdx].key] = m.inputBuf
		if m.fieldIdx == len(m.fields)-1 {
			res := m.execute()
			m.result = res
			m.isError = strings.Contains(res, `"error"`)
			m.phase = 2
			return m, nil
		}
		m.fieldIdx++
		m.inputBuf = m.values[m.fields[m.fieldIdx].key]
	case "backspace", "ctrl+h":
		if len(m.inputBuf) > 0 {
			runes := []rune(m.inputBuf)
			m.inputBuf = string(runes[:len(runes)-1])
		}
	default:
		for _, r := range msg.Runes {
			if unicode.IsPrint(r) {
				m.inputBuf += string(r)
			}
		}
	}
	return m, nil
}

func (m cryptoutilModel) updatePhase2(msg tea.KeyMsg) (cryptoutilModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.phase = 0
		m.inputBuf = ""
		m.values = make(map[string]string)
	}
	return m, nil
}

func (m cryptoutilModel) execute() string {
	ctx := context.Background()
	switch m.opIdx {
	case 0:
		return cryptoutil.Hash(ctx, cryptoutil.HashInput{
			Input:     m.values["input"],
			Algorithm: m.values["algorithm"],
			Encoding:  m.values["encoding"],
		})
	case 1:
		return cryptoutil.HMAC(ctx, cryptoutil.HMACInput{
			Key:       m.values["key"],
			Message:   m.values["message"],
			Algorithm: m.values["algorithm"],
			Encoding:  m.values["encoding"],
		})
	case 2:
		return cryptoutil.JWT(ctx, cryptoutil.JWTInput{
			Token:     m.values["token"],
			Secret:    m.values["secret"],
			Algorithm: m.values["algorithm"],
			Operation: m.values["operation"],
		})
	case 3:
		return cryptoutil.Password(ctx, cryptoutil.PasswordInput{
			Password:  m.values["password"],
			Hash:      m.values["hash"],
			Algorithm: m.values["algorithm"],
			Operation: m.values["operation"],
		})
	case 4:
		return cryptoutil.Keygen(ctx, cryptoutil.KeygenInput{
			KeyType: m.values["key_type"],
			Format:  m.values["format"],
		})
	case 5:
		return cryptoutil.Random(ctx, cryptoutil.RandomInput{
			Kind: m.values["kind"],
		})
	case 6:
		return cryptoutil.Mask(ctx, cryptoutil.MaskInput{
			Text: m.values["text"],
		})
	}
	return `{"error":"unknown operation"}`
}

func cryptoFields(opIdx int) []fieldDef {
	switch opIdx {
	case 0:
		return []fieldDef{
			{"input", "Input", true, "string to hash", false},
			{"algorithm", "Algorithm", false, "sha256 | sha512 | md5 | sha1 (default: sha256)", false},
			{"encoding", "Encoding", false, "hex | base64 (default: hex)", false},
		}
	case 1:
		return []fieldDef{
			{"key", "Key", true, "HMAC secret key", false},
			{"message", "Message", true, "message to sign", false},
			{"algorithm", "Algorithm", false, "sha256 | sha512 (default: sha256)", false},
			{"encoding", "Encoding", false, "hex | base64 (default: hex)", false},
		}
	case 2:
		return []fieldDef{
			{"operation", "Operation", true, "decode | verify | generate", false},
			{"token", "Token", false, "JWT token (for decode/verify)", false},
			{"secret", "Secret", false, "signing secret (for verify/generate)", false},
			{"algorithm", "Algorithm", false, "HS256 | HS512 (for generate)", false},
		}
	case 3:
		return []fieldDef{
			{"operation", "Operation", true, "hash | verify", false},
			{"password", "Password", true, "plaintext password", false},
			{"hash", "Hash", false, "existing hash (for verify)", false},
			{"algorithm", "Algorithm", false, "bcrypt | argon2id (default: bcrypt)", false},
		}
	case 4:
		return []fieldDef{
			{"key_type", "Key Type", true, "rsa | ec | ed25519", false},
			{"format", "Format", false, "pem | jwk (default: pem)", false},
		}
	case 5:
		return []fieldDef{
			{"kind", "Kind", true, "token | bytes | otp", false},
		}
	case 6:
		return []fieldDef{
			{"text", "Text", true, "text to scan and mask secrets", true},
		}
	}
	return nil
}

func (m cryptoutilModel) View() string {
	switch m.phase {
	case 0:
		return m.viewPhase0()
	case 1:
		return m.viewPhase1()
	case 2:
		return m.viewPhase2()
	}
	return ""
}

func (m cryptoutilModel) viewPhase0() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("◆ Security & Cryptography") + "\n\n")
	b.WriteString(normalStyle.Render("Select operation:") + "\n")
	for i, op := range cryptoOps {
		if i == m.opIdx {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", op)) + "\n")
		} else {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  %s", op)) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ navigate • Enter select • Esc back"))
	return b.String()
}

func (m cryptoutilModel) viewPhase1() string {
	if m.fieldIdx >= len(m.fields) {
		return ""
	}
	field := m.fields[m.fieldIdx]
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("◆ %s (%d/%d)", cryptoOps[m.opIdx], m.fieldIdx+1, len(m.fields))) + "\n\n")
	b.WriteString(normalStyle.Render(field.label) + "\n")
	if field.hint != "" {
		b.WriteString(dimStyle.Render(field.hint) + "\n")
	}
	b.WriteString("\n")
	if field.multiline {
		lines := strings.Split(m.inputBuf, "\n")
		start := 0
		if len(lines) > 5 {
			start = len(lines) - 5
		}
		for _, line := range lines[start:] {
			b.WriteString(boxStyle.Render(line) + "\n")
		}
		if len(m.inputBuf) == 0 {
			b.WriteString(boxStyle.Render("_") + "\n")
		}
		b.WriteString(dimStyle.Render("(ctrl+j for newline)") + "\n")
	} else {
		b.WriteString(boxStyle.Render(m.inputBuf+"_") + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("Esc back • Enter confirm"))
	return b.String()
}

func (m cryptoutilModel) viewPhase2() string {
	var b strings.Builder
	if m.isError {
		b.WriteString(errorStyle.Render("✗ Error") + "\n\n")
		b.WriteString(errorStyle.Render(m.result) + "\n")
	} else {
		b.WriteString(successStyle.Render("✓ Result") + "\n\n")
		b.WriteString(normalStyle.Render(m.result) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("Enter/Esc → back to operations"))
	return b.String()
}
