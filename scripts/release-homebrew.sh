#!/usr/bin/env bash
# release-homebrew.sh — Build binaries and upload macOS bottles to the GitHub release.
#
# Usage:
#   GITHUB_TOKEN=... bash scripts/release-homebrew.sh
#
# Prerequisites:
#   - GITHUB_TOKEN with "repo" scope (for uploading release assets)
#   - macOS (for building bottles) or Linux with macOS cross-compilation
#   - CGO_ENABLED=1 toolchain
#
# What it does:
#   1. Reads VERSION to get the version tag
#   2. Builds devforge-mcp and devforge for macOS (arm64 + x86_64)
#   3. Downloads the dpf binary for macOS
#   4. Creates tarballs: devforge-{version}.macos-arm64.tar.gz
#                                  devforge-{version}.macos-intel.tar.gz
#   5. Uploads each tarball to the GitHub release as an asset
#   6. Prints the updated formula snippet with sha256 checksums
set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
REPO="${REPO:-GustavoGutierrez/devforge-mcp}"
VERSION_FILE="VERSION"
DPF_VERSION="${DPF_VERSION:-0.2.0}"
OUTPUT_DIR="$(mktemp -d)"
ARCHIVE_DIR="$(mktemp -d)"

# Colours
RED='\033[0;31m'; YLW='\033[1;33m'; GRN='\033[0;32m'
CYN='\033[0;36m'; BLD='\033[1m'; RST='\033[0m'

# ── Helpers ───────────────────────────────────────────────────────────────────
log()  { echo -e "${BLD}${CYN}[brew-release]${RST} $*"; }
ok()   { echo -e "${GRN}  ✓${RST} $*"; }
warn() { echo -e "${YLW}  ⚠${RST} $*"; }
die()  { echo -e "${RED}  ✗ ERROR:${RST} $*" >&2; exit 1; }

need() {
  command -v "$1" &>/dev/null && return 0
  die "Missing required command: $1. Install with: brew install $1"
}

# ── Checks ───────────────────────────────────────────────────────────────────
[ -n "${GITHUB_TOKEN}" ]  || die "GITHUB_TOKEN is not set. Get one at https://github.com/settings/tokens"
[ -f "${VERSION_FILE}" ]  || die "VERSION file not found"
command -v gh             &>/dev/null || need "gh"
command -v curl           &>/dev/null || need "curl"

# Read version
VERSION="$(tr -d '[:space:]' < "${VERSION_FILE}")"
TAG="v${VERSION}"
log "Building bottles for ${BLD}v${VERSION}${RST}"

# Check tag exists
if ! gh release view "${TAG}" &>/dev/null; then
  die "Release tag ${TAG} not found. Create it first with:"
  die "  git tag v${VERSION} && git push origin v${VERSION}"
  die "  gh release create ${TAG} --title 'DevForge v${VERSION}'"
fi

# ── Build function ───────────────────────────────────────────────────────────
build_for_platform() {
  local platform="$1"   # "darwin_arm64" or "darwin_amd64"
  local out_dir="$2"

  local arch suffix
  case "${platform}" in
    darwin_arm64)  arch="arm64"; suffix="arm64" ;;
    darwin_amd64)  arch="amd64"; suffix="intel" ;;
    *) die "Unsupported platform: ${platform}" ;;
  esac

  log "Building for macOS ${arch}..."

  local build_dir="${out_dir}/build-${suffix}"
  mkdir -p "${build_dir}"

  # Build Go binaries with cross-compilation
  GOOS=darwin GOARCH="${arch}" CGO_ENABLED=1 \
    go build -ldflags="-s -w" -o "${build_dir}/devforge-mcp" ./cmd/devforge-mcp/
  GOOS=darwin GOARCH="${arch}" CGO_ENABLED=1 \
    go build -ldflags="-s -w" -o "${build_dir}/devforge"     ./cmd/devforge/

  # Download dpf for the target architecture
  local dpf_url="https://github.com/GustavoGutierrez/devpixelforge/releases/download/v${DPF_VERSION}/dpf-${DPF_VERSION}-macos.tar.gz"
  local dpf_archive="${build_dir}/dpf.tar.gz"
  curl -sSL --fail -o "${dpf_archive}" "${dpf_url}" \
    || warn "dpf download failed — dpf will not be included for ${suffix}"

  if [ -f "${dpf_archive}" ]; then
    tar -xzf "${dpf_archive}" -C "${build_dir}/"
    rm -f "${dpf_archive}"
    chmod +x "${build_dir}/dpf" 2>/dev/null || true
  fi

  # Create archive
  local archive="${ARCHIVE_DIR}/devforge-${VERSION}.macos-${suffix}.tar.gz"
  tar -czf "${archive}" -C "${out_dir}" "$(basename "${build_dir}")"
  ok "Created ${archive}"

  # Compute sha256
  local sha256
  sha256="$(shasum -a 256 "${archive}" | cut -d' ' -f1)"
  echo "${suffix}:${sha256}" >> "${OUTPUT_DIR}/checksums.txt"

  # Upload to GitHub release
  log "Uploading ${archive}..."
  gh release upload "${TAG}" "${archive}" \
    --clobber \
    || warn "Upload failed — asset may already exist or token lacks permissions"
}

# ── Main ─────────────────────────────────────────────────────────────────────
cd "$(dirname "$0")/.."

build_for_platform "darwin_arm64"  "${OUTPUT_DIR}"
build_for_platform "darwin_amd64" "${OUTPUT_DIR}"

# ── Print formula snippet ───────────────────────────────────────────────────
log "Bottles uploaded. Add these checksums to Formula/devforge.rb:"
echo ""
while IFS=: read -r suffix sha256; do
  case "${suffix}" in
    arm64)
      echo "    on_macos_apple do"
      echo "      url \"https://github.com/${REPO}/releases/download/${TAG}/devforge-${VERSION}.macos-arm64.tar.gz\""
      echo "      sha256 \"${sha256}\""
      echo "    end"
      ;;
    intel)
      echo "    on_macos_sequential do"
      echo "      url \"https://github.com/${REPO}/releases/download/${TAG}/devforge-${VERSION}.macos-intel.tar.gz\""
      echo "      sha256 \"${sha256}\""
      echo "    end"
      ;;
  esac
done < "${OUTPUT_DIR}/checksums.txt"
echo ""

# Cleanup
rm -rf "${OUTPUT_DIR}" "${ARCHIVE_DIR}"

ok "Done! Update Formula/devforge.rb with the checksums above, then:"
ok "  git add Formula/devforge.rb && git commit -m 'chore(homebrew): update bottles for v${VERSION}'"
