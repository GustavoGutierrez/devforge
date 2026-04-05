#!/usr/bin/env bash
# install.sh — Install devforge to ~/.local/share/devforge/versions/X.Y.Z/
# and create symlinks in ~/.local/bin/
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# ── Version ────────────────────────────────────────────────────────────────────
VERSION_FILE="${PROJECT_DIR}/VERSION"
if [ -f "$VERSION_FILE" ]; then
    VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
else
    VERSION="0.0.0"
fi

SHARE_BASE="${HOME}/.local/share/devforge"
SHARE_DIR="${SHARE_BASE}/versions/${VERSION}"
LINK_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/devforge"
CONFIG_FILE="${CONFIG_DIR}/config.json"
DIST_DIR="${PROJECT_DIR}/dist"

RED='\033[0;31m'
YLW='\033[1;33m'
GRN='\033[0;32m'
CYN='\033[0;36m'
BLD='\033[1m'
RST='\033[0m'

echo ""
echo -e "${BLD}DevForge — Installer v${VERSION}${RST}"
echo -e "${CYN}──────────────────────────────────────────${RST}"
echo ""

# ── Build ──────────────────────────────────────────────────────────────────────
echo -e "${BLD}Step 1/4 — Building binaries...${RST}"
cd "$PROJECT_DIR"
go build -o "${DIST_DIR}/devforge-mcp" ./cmd/devforge-mcp/
go build -o "${DIST_DIR}/devforge"     ./cmd/devforge/
echo -e "  ${GRN}✓${RST} devforge-mcp"
echo -e "  ${GRN}✓${RST} devforge"

# ── Create share dir and copy files ───────────────────────────────────────────
echo ""
echo -e "${BLD}Step 2/4 — Installing to ${SHARE_DIR}...${RST}"
mkdir -p "${SHARE_DIR}"

rm -f "${SHARE_DIR}/devforge-mcp" "${SHARE_DIR}/devforge"
cp "${DIST_DIR}/devforge-mcp" "${SHARE_DIR}/devforge-mcp"
cp "${DIST_DIR}/devforge"     "${SHARE_DIR}/devforge"
chmod +x "${SHARE_DIR}/devforge-mcp" "${SHARE_DIR}/devforge"
echo -e "  ${GRN}✓${RST} devforge-mcp"
echo -e "  ${GRN}✓${RST} devforge"

if [ -f "${DIST_DIR}/dpf" ]; then
    rm -f "${SHARE_DIR}/dpf"
    cp "${DIST_DIR}/dpf" "${SHARE_DIR}/dpf"
    chmod +x "${SHARE_DIR}/dpf"
    echo -e "  ${GRN}✓${RST} dpf"
elif [ -f "${PROJECT_DIR}/bin/dpf" ]; then
    cp "${PROJECT_DIR}/bin/dpf" "${SHARE_DIR}/dpf"
    chmod +x "${SHARE_DIR}/dpf"
    echo -e "  ${GRN}✓${RST} dpf (from bin/)"
else
    echo -e "  ${YLW}⚠${RST}  dpf not found — media tools unavailable"
    echo -e "  ${YLW}→${RST}  Run: bash scripts/install-dpf.sh"
fi

# Update 'current' symlink
ln -sfn "versions/${VERSION}" "${SHARE_BASE}/current"
echo -e "  ${GRN}✓${RST} ${SHARE_BASE}/current -> versions/${VERSION}"

# ── Symlinks in ~/.local/bin ───────────────────────────────────────────────────
echo ""
echo -e "${BLD}Step 3/4 — Creating symlinks in ${LINK_DIR}...${RST}"
mkdir -p "${LINK_DIR}"

for BIN in devforge-mcp devforge; do
    TARGET="${SHARE_DIR}/${BIN}"
    LINK="${LINK_DIR}/${BIN}"
    if [ -f "${TARGET}" ]; then
        ln -sf "${TARGET}" "${LINK}"
        echo -e "  ${GRN}✓${RST} ${LINK} -> ${TARGET}"
    fi
done

if [ -f "${SHARE_DIR}/dpf" ]; then
    ln -sf "${SHARE_DIR}/dpf" "${LINK_DIR}/dpf"
    echo -e "  ${GRN}✓${RST} ${LINK_DIR}/dpf -> ${SHARE_DIR}/dpf"
fi

# ── Initial config ─────────────────────────────────────────────────────────────
echo ""
echo -e "${BLD}Step 4/4 — Configuration${RST}"
mkdir -p "${CONFIG_DIR}"
if [ ! -f "${CONFIG_FILE}" ]; then
    cat > "${CONFIG_FILE}" <<'EOF'
{
  "gemini_api_key": "",
  "image_model": "gemini-2.5-flash-image"
}
EOF
    chmod 600 "${CONFIG_FILE}"
    echo -e "  ${GRN}✓${RST} Created ${CONFIG_FILE}"
else
    echo -e "  ${YLW}kept${RST}   ${CONFIG_FILE} (already exists)"
fi

# ── Summary ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${GRN}${BLD}DevForge v${VERSION} installed successfully.${RST}"
echo ""
echo -e "${BLD}Locations:${RST}"
echo "  Binaries : ${SHARE_DIR}/"
echo "  Config   : ${CONFIG_FILE}"
echo "  Symlinks : ${LINK_DIR}/devforge-mcp, devforge"
echo ""
echo "Ensure ${LINK_DIR} is in your PATH:"
echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
echo ""
