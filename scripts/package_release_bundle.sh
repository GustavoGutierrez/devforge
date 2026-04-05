#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

VERSION=""
OUTPUT_DIR="${PROJECT_DIR}/dist/release"

usage() {
  cat <<'EOF'
Usage: bash scripts/package_release_bundle.sh --version X.Y.Z [--output-dir path]

Builds the canonical Linux amd64 DevForge release bundle containing:
  - devforge
  - devforge-mcp
  - dpf
  - devforge.db
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

DPF_SOURCE="${PROJECT_DIR}/bin/dpf"
if [ ! -x "$DPF_SOURCE" ]; then
  echo "Expected executable dpf binary at ${DPF_SOURCE}. Run bash scripts/install-dpf.sh and chmod +x bin/dpf." >&2
  exit 1
fi

RELEASE_ROOT="${OUTPUT_DIR}/devforge_${VERSION}_linux_amd64"
ARCHIVE_NAME="devforge_${VERSION}_linux_amd64.tar.gz"
ARCHIVE_PATH="${OUTPUT_DIR}/${ARCHIVE_NAME}"

rm -rf "$RELEASE_ROOT"
mkdir -p "$RELEASE_ROOT"
mkdir -p "$OUTPUT_DIR"

pushd "$PROJECT_DIR" >/dev/null

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${RELEASE_ROOT}/devforge-mcp" ./cmd/devforge-mcp/
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "${RELEASE_ROOT}/devforge" ./cmd/devforge/

cp "$DPF_SOURCE" "${RELEASE_ROOT}/dpf"
chmod +x "${RELEASE_ROOT}/dpf"

CGO_ENABLED=1 go run ./scripts/init_db_runner -db "${RELEASE_ROOT}/devforge.db"

SEED_ARGS=()
for seed_file in db/seeds/*.sql; do
  if [ -f "$seed_file" ]; then
    SEED_ARGS+=( -sql "$seed_file" )
  fi
done

if [ "${#SEED_ARGS[@]}" -gt 0 ]; then
  CGO_ENABLED=1 go run ./scripts/seed_runner -db "${RELEASE_ROOT}/devforge.db" "${SEED_ARGS[@]}"
fi

tar -C "$RELEASE_ROOT" -czf "$ARCHIVE_PATH" .

popd >/dev/null

echo "Created ${ARCHIVE_PATH}"
