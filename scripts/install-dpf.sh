#!/usr/bin/env bash
# install-dpf.sh — Download the pre-built DevPixelForge binary
# Usage:
#   bash scripts/install-dpf.sh
#   bash scripts/install-dpf.sh v0.4.3
#   bash scripts/install-dpf.sh --os linux --arch amd64
#   bash scripts/install-dpf.sh v0.4.3 --os darwin --arch arm64 --output /tmp/dpf
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="${PROJECT_DIR}/bin"
DEFAULT_OUTPUT="${BIN_DIR}/dpf"
REPO="GustavoGutierrez/devpixelforge"

if [[ -t 1 ]]; then
    GRN='\033[0;32m'; YLW='\033[1;33m'; RED='\033[0;31m'; RST='\033[0m'
else
    GRN=''; YLW=''; RED=''; RST=''
fi

info()    { echo -e "${GRN}→${RST} $*"; }
warn()    { echo -e "${YLW}!${RST} $*"; }
error()   { echo -e "${RED}✗${RST} $*" >&2; }

fetch_latest() {
    local tag
    tag=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"v\?\([^"]*\)".*/\1/')
    if [ -z "$tag" ]; then
        error "Could not fetch latest release tag from GitHub API."
        exit 1
    fi
    echo "$tag"
}

usage() {
    cat <<'EOF'
Usage: bash scripts/install-dpf.sh [version] [--os linux|darwin] [--arch amd64|arm64] [--output path]

Defaults:
  - version: latest release from GustavoGutierrez/devpixelforge
  - os/arch: detected from the current machine
  - output: ./bin/dpf

Supported assets:
  - linux/amd64  -> dpf-linux-amd64.tar.gz
  - darwin/arm64 -> dpf-macos-arm64.tar.gz
EOF
}

resolve_target() {
    if [ -n "$TARGET_OS" ] && [ -n "$TARGET_ARCH" ]; then
        return 0
    fi

    case "$(uname -s)" in
        Linux)
            TARGET_OS="linux"
            ;;
        Darwin)
            TARGET_OS="darwin"
            ;;
        *)
            error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)
            TARGET_ARCH="amd64"
            ;;
        arm64|aarch64)
            TARGET_ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
}

download_dpf() {
    local tag="$1"
    local output_path="$2"
    local platform_key="${TARGET_OS}/${TARGET_ARCH}"
    local archive_name=""
    # Known binary name inside the DevPixelForge release archive:
    #   dpf-linux-amd64.tar.gz  → binary is named: dpf-dpf-linux-amd64
    #   dpf-macos-arm64.tar.gz  → binary is named: dpf-dpf-macos-arm64
    local dpf_binary_name=""
    local url=""
    local tmp_dir=""
    local archive_path=""

    case "$platform_key" in
        linux/amd64)
            archive_name="dpf-linux-amd64.tar.gz"
            dpf_binary_name="dpf-dpf-linux-amd64"
            ;;
        darwin/arm64)
            archive_name="dpf-macos-arm64.tar.gz"
            dpf_binary_name="dpf-dpf-macos-arm64"
            ;;
        *)
            error "Unsupported target combination: ${platform_key}"
            exit 1
            ;;
    esac

    url="https://github.com/${REPO}/releases/download/${tag}/${archive_name}"

    mkdir -p "$(dirname "${output_path}")"
    tmp_dir="$(mktemp -d)"
    archive_path="${tmp_dir}/${archive_name}"

    trap 'rm -rf "${tmp_dir}"' RETURN

    info "Downloading DevPixelForge ${tag}..."
    info "From: ${url}"

    if command -v curl &>/dev/null; then
        curl -fsSL -o "${archive_path}" "$url"
    elif command -v wget &>/dev/null; then
        wget -q -O "${archive_path}" "$url"
    else
        error "Neither curl nor wget found — cannot download."
        exit 1
    fi

    tar -xzf "${archive_path}" -C "${tmp_dir}"

    # Search order: known DevPixelForge binary name first, then generic dpf fallbacks.
    # DevPixelForge archives produce a binary named dpf-dpf-{platform} (e.g. dpf-dpf-linux-amd64).
    shopt -s nullglob
    candidates=()
    for candidate in \
        "${tmp_dir}/${dpf_binary_name}" \
        "${tmp_dir}/dpf" \
        "${tmp_dir}"/dpf-* \
        "${tmp_dir}"/*/dpf \
        "${tmp_dir}"/*/dpf-*; do
        if [ -f "${candidate}" ]; then
            candidates+=("${candidate}")
        fi
    done
    shopt -u nullglob

    if [ "${#candidates[@]}" -eq 0 ]; then
        error "Downloaded archive did not contain a dpf binary."
        error "Expected binary name: ${dpf_binary_name}"
        exit 1
    fi

    info "Found binary: ${candidates[0]} → renaming to dpf"
    mv "${candidates[0]}" "${output_path}"
    chmod +x "${output_path}"
    info "Installed: ${output_path}"
}

VERSION=""
TARGET_OS="${DPF_TARGET_OS:-}"
TARGET_ARCH="${DPF_TARGET_ARCH:-}"
OUTPUT_PATH="${DPF_OUTPUT:-${DEFAULT_OUTPUT}}"

while [ $# -gt 0 ]; do
    case "$1" in
        --os)
            TARGET_OS="$2"
            shift 2
            ;;
        --arch)
            TARGET_ARCH="$2"
            shift 2
            ;;
        --output)
            OUTPUT_PATH="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            if [ -z "$VERSION" ]; then
                VERSION="$1"
                shift
            else
                error "Unknown argument: $1"
                usage >&2
                exit 1
            fi
            ;;
    esac
done

resolve_target

if [ -z "$VERSION" ]; then
    info "Fetching latest release tag..."
    VERSION=$(fetch_latest)
    info "Latest version: v${VERSION}"
fi

VERSION="${VERSION#v}"
TAG="v${VERSION}"

download_dpf "${TAG}" "${OUTPUT_PATH}"

echo ""
info "Done. Run 'chmod +x ${OUTPUT_PATH}' to ensure it is executable."
info "See docs/install.md for the full installation guide."
