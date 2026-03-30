#!/usr/bin/env bash
# setup-mcp-client.sh — Configure devforge-mcp in your MCP client of choice.
#
# Supports: VS Code (Copilot), Claude Desktop, Claude Code, OpenCode
# Run with no arguments for an interactive menu.
set -euo pipefail

# ── Paths ─────────────────────────────────────────────────────────────────────
# Detect devforge-mcp binary with priority:
# 1. ENV override
# 2. Homebrew paths (Linuxbrew and macOS)
# 3. which command (user's PATH)
# 4. Common installation directories
detect_binary() {
    local path

    # Try Homebrew paths first (most common for devforge-mcp installations)
    local homebrew_paths=(
        "/home/linuxbrew/.linuxbrew/bin/devforge-mcp"  # Linuxbrew (Linux)
        "/opt/homebrew/bin/devforge-mcp"               # Homebrew ARM (macOS Apple Silicon)
        "/usr/local/bin/devforge-mcp"                  # Homebrew Intel (macOS)
    )

    for path in "${homebrew_paths[@]}"; do
        if [ -f "$path" ]; then
            echo "$path"
            return 0
        fi
    done

    # Try which to find in PATH
    if path=$(command -v devforge-mcp 2>/dev/null) && [ -n "$path" ]; then
        echo "$path"
        return 0
    fi

    # Common installation directories
    local local_paths=(
        "${HOME}/.local/bin/devforge-mcp"
        "${HOME}/.cargo/bin/devforge-mcp"
        "${HOME}/go/bin/devforge-mcp"
    )

    for path in "${local_paths[@]}"; do
        if [ -f "$path" ]; then
            echo "$path"
            return 0
        fi
    done

    # Fallback to default
    echo "${HOME}/.local/bin/devforge-mcp"
}

# Detect config.json path - prefer XDG_CONFIG_HOME, fallback to ~/.config
detect_config() {
    local xdg_config="${XDG_CONFIG_HOME:-${HOME}/.config}"
    local homebrew_config=""

    # Try Homebrew-specific config location (for Linuxbrew/macOS Homebrew installs)
    local homebrew_lib_paths=(
        "/home/linuxbrew/.linuxbrew/etc/devforge/config.json"  # Linuxbrew
        "/opt/homebrew/etc/devforge/config.json"              # macOS ARM
        "/usr/local/etc/devforge/config.json"                 # macOS Intel
    )

    for path in "${homebrew_lib_paths[@]}"; do
        if [ -f "$path" ]; then
            echo "$path"
            return 0
        fi
    done

    # Standard XDG/config location
    echo "${xdg_config}/devforge/config.json"
}

BIN="$(detect_binary)"
CFG="$(detect_config)"
BIN="${DEV_FORGE_BIN:-$BIN}"
CFG="${DEV_FORGE_CONFIG:-$CFG}"

# ── Colors ────────────────────────────────────────────────────────────────────
if [ -t 1 ]; then
    BOLD='\033[1m'; CYAN='\033[0;36m'; GREEN='\033[0;32m'
    YELLOW='\033[0;33m'; RED='\033[0;31m'; RESET='\033[0m'
else
    BOLD=''; CYAN=''; GREEN=''; YELLOW=''; RED=''; RESET=''
fi

# ── Helpers ───────────────────────────────────────────────────────────────────
info()    { echo -e "${CYAN}→${RESET} $*"; }
success() { echo -e "${GREEN}✓${RESET} $*"; }
warn()    { echo -e "${YELLOW}!${RESET} $*"; }
error()   { echo -e "${RED}✗${RESET} $*" >&2; }
header()  { echo -e "\n${BOLD}$*${RESET}"; }

check_binary() {
    if [ ! -f "$BIN" ]; then
        error "devforge-mcp binary not found at: $BIN"
        error "Run 'bash scripts/install.sh' first."
        exit 1
    fi
}

# Merge a JSON fragment into an existing JSON file using jq (if available),
# otherwise append a comment warning and write the raw block.
# Usage: merge_json_key <file> <jq-filter> <fallback-content>
merge_json() {
    local file="$1"
    local filter="$2"      # jq filter to merge, e.g.: .mcpServers["devforge"] = {...}
    local fallback="$3"    # content to write if jq is not available

    if command -v jq &>/dev/null; then
        if [ -f "$file" ] && [ -s "$file" ]; then
            local tmp
            tmp="$(mktemp)"
            jq "$filter" "$file" > "$tmp" && mv "$tmp" "$file"
        else
            echo '{}' | jq "$filter" > "$file"
        fi
    else
        warn "jq not found — writing config from scratch (existing entries may be lost)."
        warn "Install jq for safe merging: sudo apt-get install jq"
        echo "$fallback" > "$file"
    fi
}

# ── Client configurators ──────────────────────────────────────────────────────

setup_vscode() {
    header "Configuring VS Code (GitHub Copilot MCP)"

    # Default to current directory; user can choose global settings instead
    local scope
    echo "Install in:"
    echo "  1) Current workspace (.vscode/mcp.json) — recommended"
    echo "  2) User settings (~/.config/Code/User/mcp.json)"
    read -rp "Choice [1]: " scope
    scope="${scope:-1}"

    local target
    case "$scope" in
        2) target="${HOME}/.config/Code/User/mcp.json" ;;
        *) target=".vscode/mcp.json"; mkdir -p .vscode ;;
    esac

    local entry
    entry=$(printf '{"type":"stdio","command":"%s","args":[],"env":{"DEV_FORGE_CONFIG":"%s"}}' "$BIN" "$CFG")

    local filter
    filter=$(printf '.servers["devforge"] = %s' "$entry")

    local fallback
    fallback=$(printf '{\n  "servers": {\n    "devforge": {\n      "type": "stdio",\n      "command": "%s",\n      "args": [],\n      "env": {\n        "DEV_FORGE_CONFIG": "%s"\n      }\n    }\n  }\n}' "$BIN" "$CFG")

    merge_json "$target" "$filter" "$fallback"
    success "Written to $target"
    info "Reload VS Code (Ctrl+Shift+P → 'Developer: Reload Window') to activate."
}

setup_claude_desktop() {
    header "Configuring Claude Desktop"

    local config_file
    case "$(uname -s)" in
        Darwin) config_file="${HOME}/Library/Application Support/Claude/claude_desktop_config.json" ;;
        Linux)  config_file="${HOME}/.config/Claude/claude_desktop_config.json" ;;
        *)
            error "Unsupported OS: $(uname -s)"
            error "Edit the config file manually — see docs/mcp-connect.md"
            return 1
            ;;
    esac

    mkdir -p "$(dirname "$config_file")"

    local entry
    entry=$(printf '{"command":"%s","args":[],"env":{"DEV_FORGE_CONFIG":"%s"}}' "$BIN" "$CFG")

    local filter
    filter=$(printf '.mcpServers["devforge"] = %s' "$entry")

    local fallback
    fallback=$(printf '{\n  "mcpServers": {\n    "devforge": {\n      "command": "%s",\n      "args": [],\n      "env": {\n        "DEV_FORGE_CONFIG": "%s"\n      }\n    }\n  }\n}' "$BIN" "$CFG")

    merge_json "$config_file" "$filter" "$fallback"
    success "Written to $config_file"
    info "Restart Claude Desktop to activate."
}

setup_claude_code() {
    header "Configuring Claude Code"

    echo "Install as:"
    echo "  1) Global user config (applies to every project)"
    echo "  2) Project-level .mcp.json (current directory)"
    read -rp "Choice [1]: " scope
    scope="${scope:-1}"

    case "$scope" in
        2)
            local entry
            entry=$(printf '{"command":"%s","args":[],"env":{"DEV_FORGE_CONFIG":"%s"}}' "$BIN" "$CFG")
            local filter
            filter=$(printf '.mcpServers["devforge"] = %s' "$entry")
            local fallback
            fallback=$(printf '{\n  "mcpServers": {\n    "devforge": {\n      "command": "%s",\n      "args": [],\n      "env": {\n        "DEV_FORGE_CONFIG": "%s"\n      }\n    }\n  }\n}' "$BIN" "$CFG")
            merge_json ".mcp.json" "$filter" "$fallback"
            success "Written to .mcp.json"
            ;;
        *)
            if command -v claude &>/dev/null; then
                claude mcp add devforge "$BIN" -e "DEV_FORGE_CONFIG=$CFG"
                success "Registered via 'claude mcp add'."
                info "Verify with: claude mcp list"
            else
                warn "'claude' CLI not found — writing to ~/.claude.json manually."
                local target="${HOME}/.claude.json"
                local entry
                entry=$(printf '{"command":"%s","args":[],"env":{"DEV_FORGE_CONFIG":"%s"}}' "$BIN" "$CFG")
                local filter
                filter=$(printf '.mcpServers["devforge"] = %s' "$entry")
                local fallback
                fallback=$(printf '{\n  "mcpServers": {\n    "devforge": {\n      "command": "%s",\n      "args": [],\n      "env": {\n        "DEV_FORGE_CONFIG": "%s"\n      }\n    }\n  }\n}' "$BIN" "$CFG")
                merge_json "$target" "$filter" "$fallback"
                success "Written to $target"
            fi
            ;;
    esac
    info "Start a new 'claude' session to activate."
}

setup_opencode() {
    header "Configuring OpenCode"

    echo "Install as:"
    echo "  1) Global user config (~/.config/opencode/config.json)"
    echo "  2) Project-local config (opencode.json in current directory)"
    read -rp "Choice [1]: " scope
    scope="${scope:-1}"

    local target
    case "$scope" in
        2) target="opencode.json" ;;
        *) target="${HOME}/.config/opencode/config.json"; mkdir -p "${HOME}/.config/opencode" ;;
    esac

    local entry
    entry=$(printf '{"type":"local","command":["%s"],"environment":{"DEV_FORGE_CONFIG":"%s"}}' "$BIN" "$CFG")

    local filter
    filter=$(printf '.mcp["devforge"] = %s' "$entry")

    # Seed the $schema key if file is new
    local fallback
    fallback=$(printf '{\n  "$schema": "https://opencode.ai/config.json",\n  "mcp": {\n    "devforge": {\n      "type": "local",\n      "command": ["%s"],\n      "environment": {\n        "DEV_FORGE_CONFIG": "%s"\n      }\n    }\n  }\n}' "$BIN" "$CFG")

    # Preserve $schema when merging with jq
    if command -v jq &>/dev/null && [ -f "$target" ] && [ -s "$target" ]; then
        local tmp
        tmp="$(mktemp)"
        jq "$filter" "$target" > "$tmp" && mv "$tmp" "$target"
    else
        echo "$fallback" > "$target"
    fi

    success "Written to $target"
    info "Run 'opencode' in your terminal to activate."
}

# ── Preflight check ───────────────────────────────────────────────────────────

print_status() {
    header "devforge-mcp setup"
    echo ""
    if [ -f "$BIN" ]; then
        success "Binary:  $BIN"
    else
        error   "Binary:  $BIN  (NOT FOUND — run scripts/install.sh first)"
    fi
    if [ -f "$CFG" ]; then
        success "Config:  $CFG"
    else
        warn    "Config:  $CFG  (not found — will be created on first use)"
    fi
    echo ""
}

# ── Main menu ─────────────────────────────────────────────────────────────────

print_status
check_binary

echo "Which MCP client do you want to configure?"
echo ""
echo "  1) VS Code — GitHub Copilot"
echo "  2) Claude Desktop"
echo "  3) Claude Code (CLI)"
echo "  4) OpenCode"
echo "  5) All of the above"
echo ""
read -rp "Choice [1]: " choice
choice="${choice:-1}"

case "$choice" in
    1) setup_vscode ;;
    2) setup_claude_desktop ;;
    3) setup_claude_code ;;
    4) setup_opencode ;;
    5)
        setup_vscode
        setup_claude_desktop
        setup_claude_code
        setup_opencode
        ;;
    *)
        error "Invalid choice: $choice"
        exit 1
        ;;
esac

echo ""
success "Done. See docs/mcp-connect.md for usage examples."
