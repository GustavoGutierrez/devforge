#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

VERSION=""
OUTPUT_DIR="${PROJECT_DIR}/dist/release"
TARGET_OS="linux"
TARGET_ARCH="amd64"

usage() {
  cat <<'EOF'
Usage: bash scripts/package_release_bundle.sh --version X.Y.Z [--target-os linux|darwin] [--target-arch amd64|arm64] [--output-dir path]

Builds a DevForge runtime bundle containing:
  - devforge
  - devforge-mcp
  - dpf

Supported target combinations:
  - linux/amd64
  - darwin/arm64
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --target-os)
      TARGET_OS="$2"
      shift 2
      ;;
    --target-arch)
      TARGET_ARCH="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -z "$VERSION" ]; then
  echo "--version is required" >&2
  usage >&2
  exit 1
fi

VERSION_FILE="${PROJECT_DIR}/VERSION"
if [ ! -f "$VERSION_FILE" ]; then
  echo "VERSION file not found at ${VERSION_FILE}" >&2
  exit 1
fi

CURRENT_VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
if [ "$CURRENT_VERSION" != "$VERSION" ]; then
  echo "VERSION file (${CURRENT_VERSION}) does not match requested bundle version (${VERSION})" >&2
  exit 1
fi

case "${TARGET_OS}/${TARGET_ARCH}" in
  linux/amd64)
    PLATFORM_SUFFIX="linux_amd64"
    ;;
  darwin/arm64)
    PLATFORM_SUFFIX="darwin_arm64"
    ;;
  *)
    echo "Unsupported target combination: ${TARGET_OS}/${TARGET_ARCH}" >&2
    exit 1
    ;;
esac

RELEASE_ROOT="${OUTPUT_DIR}/devforge_${VERSION}_${PLATFORM_SUFFIX}"
ARCHIVE_NAME="devforge_${VERSION}_${PLATFORM_SUFFIX}.tar.gz"
ARCHIVE_PATH="${OUTPUT_DIR}/${ARCHIVE_NAME}"

rm -rf "$RELEASE_ROOT"
mkdir -p "$RELEASE_ROOT"
mkdir -p "$OUTPUT_DIR"

pushd "$PROJECT_DIR" >/dev/null

GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" go build -ldflags="-s -w" -o "${RELEASE_ROOT}/devforge-mcp" ./cmd/devforge-mcp/
GOOS="${TARGET_OS}" GOARCH="${TARGET_ARCH}" go build -ldflags="-s -w" -o "${RELEASE_ROOT}/devforge" ./cmd/devforge/

bash scripts/install-dpf.sh \
  --os "${TARGET_OS}" \
  --arch "${TARGET_ARCH}" \
  --output "${RELEASE_ROOT}/dpf"

tar -C "$RELEASE_ROOT" -czf "$ARCHIVE_PATH" .

popd >/dev/null

echo "Created ${ARCHIVE_PATH}"
