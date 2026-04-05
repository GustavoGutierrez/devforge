#!/usr/bin/env bash
# uninstall.sh — Remove devforge installation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

VERSION_FILE="${PROJECT_DIR}/VERSION"
VERSION="$(tr -d '[:space:]' < "$VERSION_FILE" 2>/dev/null || echo "")"

SHARE_BASE="${HOME}/.local/share/devforge"
LINK_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/devforge"

RED='\033[0;31m'
YLW='\033[1;33m'
GRN='\033[0;32m'
CYN='\033[0;36m'
BLD='\033[1m'
RST='\033[0m'

echo ""
echo -e "${BLD}DevForge — Uninstaller${RST}"
echo -e "${RED}──────────────────────────────────────────${RST}"
echo ""

# ── Detect installed versions ─────────────────────────────────────────────────
VERSIONS_DIR="${SHARE_BASE}/versions"
INSTALLED_VERSIONS=()
if [ -d "${VERSIONS_DIR}" ]; then
    while IFS= read -r -d '' d; do
        INSTALLED_VERSIONS+=("$(basename "$d")")
    done < <(find "${VERSIONS_DIR}" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null | sort -z)
fi

if [ ${#INSTALLED_VERSIONS[@]} -eq 0 ]; then
    echo -e "${YLW}No installed versions found in ${VERSIONS_DIR}${RST}"
    SHARE_DIR=""
else
    echo -e "Installed versions:"
    for v in "${INSTALLED_VERSIONS[@]}"; do
        echo "  - ${v}"
    done
    echo ""
    echo "This will remove:"
    echo -e "  ${CYN}Share directory${RST}  ${SHARE_BASE}/"
    echo -e "  ${CYN}Symlinks${RST}         ${LINK_DIR}/devforge-mcp, devforge, dpf"
fi
echo -e "  ${CYN}Config${RST}           ${CONFIG_DIR}/ (asked separately)"
echo ""

# ── Final confirmation ─────────────────────────────────────────────────────────
echo -n "Proceed with uninstall? [y/N] "
read -r CONFIRM
if [[ ! "${CONFIRM}" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi
echo ""

# ── Remove symlinks ────────────────────────────────────────────────────────────
for BIN in devforge-mcp devforge dpf; do
    LINK="${LINK_DIR}/${BIN}"
    if [ -L "${LINK}" ]; then
        rm -f "${LINK}"
        echo -e "  ${GRN}removed${RST}  symlink ${LINK}"
    elif [ -f "${LINK}" ]; then
        # Could be a plain binary from old install
        echo -e "  ${YLW}found${RST}    plain file ${LINK} — not removed (not a symlink)"
    else
        echo -e "  ${YLW}not found${RST} ${LINK}"
    fi
done

# ── Remove share directory ─────────────────────────────────────────────────────
if [ -d "${SHARE_BASE}" ]; then
    rm -rf "${SHARE_BASE}"
    echo -e "  ${GRN}removed${RST}  ${SHARE_BASE}/"
else
    echo -e "  ${YLW}not found${RST} ${SHARE_BASE}/"
fi

# ── Remove config directory (optional) ────────────────────────────────────────
if [ -d "${CONFIG_DIR}" ]; then
    echo ""
    echo -n "Remove config directory ${CONFIG_DIR}? [y/N] "
    read -r REMOVE_CONFIG
    if [[ "${REMOVE_CONFIG}" =~ ^[Yy]$ ]]; then
        rm -rf "${CONFIG_DIR}"
        echo -e "  ${GRN}removed${RST}  ${CONFIG_DIR}/"
    else
        echo -e "  ${YLW}kept${RST}     ${CONFIG_DIR}/"
    fi
else
    echo -e "  ${YLW}not found${RST} ${CONFIG_DIR}/"
fi

echo ""
echo -e "${GRN}${BLD}DevForge uninstalled.${RST}"
echo ""
