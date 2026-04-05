#!/usr/bin/env bash
set -euo pipefail

cat <<'EOF' >&2
scripts/release-homebrew.sh is deprecated.

Use the canonical Linux-first release flow instead:
  - .github/workflows/release.yml
  - scripts/package_release_bundle.sh
  - packaging/homebrew/Formula/devforge.rb

For an existing tag rerun:
  gh workflow run release.yml -f tag=vX.Y.Z --repo GustavoGutierrez/devforge
EOF

exit 1
