#!/usr/bin/env bash
#
# Copy ARCHITECTURE.md files from the repo into docs/architecture/
# for VitePress rendering. Rewrites cross-links between architecture files.
#
# This runs as part of `make docs` — the output files are .gitignored.

set -e

# Resolve paths relative to the repo root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCS_ARCH_DIR="$SCRIPT_DIR/architecture"

mkdir -p "$DOCS_ARCH_DIR"

# Map of source ARCHITECTURE.md (relative to repo root) -> output basename
declare -A ARCH_FILES=(
  ["ARCHITECTURE.md"]="system"
  ["server/ARCHITECTURE.md"]="server"
  ["client/ARCHITECTURE.md"]="client"
  ["plik/ARCHITECTURE.md"]="library"
  ["webapp/ARCHITECTURE.md"]="webapp"
  ["testing/ARCHITECTURE.md"]="testing"
  ["releaser/ARCHITECTURE.md"]="releaser"
  [".github/ARCHITECTURE.md"]="github"
)

for src in "${!ARCH_FILES[@]}"; do
  name="${ARCH_FILES[$src]}"
  out="$DOCS_ARCH_DIR/${name}.md"
  full_src="$REPO_ROOT/$src"

  if [[ ! -f "$full_src" ]]; then
    echo "WARNING: $src not found, skipping"
    continue
  fi

  # Rewrite cross-links between ARCHITECTURE.md files to VitePress paths.
  # Preserves any #anchor or query string after the filename.
  # Pattern: ](path/ARCHITECTURE.md...) -> ](./name.md...)
  sed -E \
    -e "s|\]\(\.\.\/ARCHITECTURE\.md([)#])|\](./system.md\1|g" \
    -e "s|\]\(ARCHITECTURE\.md([)#])|\](./system.md\1|g" \
    -e "s|\]\(server/ARCHITECTURE\.md([)#])|\](./server.md\1|g" \
    -e "s|\]\(\.\./server/ARCHITECTURE\.md([)#])|\](./server.md\1|g" \
    -e "s|\]\(client/ARCHITECTURE\.md([)#])|\](./client.md\1|g" \
    -e "s|\]\(\.\./client/ARCHITECTURE\.md([)#])|\](./client.md\1|g" \
    -e "s|\]\(plik/ARCHITECTURE\.md([)#])|\](./library.md\1|g" \
    -e "s|\]\(\.\./plik/ARCHITECTURE\.md([)#])|\](./library.md\1|g" \
    -e "s|\]\(webapp/ARCHITECTURE\.md([)#])|\](./webapp.md\1|g" \
    -e "s|\]\(\.\./webapp/ARCHITECTURE\.md([)#])|\](./webapp.md\1|g" \
    -e "s|\]\(testing/ARCHITECTURE\.md([)#])|\](./testing.md\1|g" \
    -e "s|\]\(\.\./testing/ARCHITECTURE\.md([)#])|\](./testing.md\1|g" \
    -e "s|\]\(releaser/ARCHITECTURE\.md([)#])|\](./releaser.md\1|g" \
    -e "s|\]\(\.\./releaser/ARCHITECTURE\.md([)#])|\](./releaser.md\1|g" \
    -e "s|\]\(\.github/ARCHITECTURE\.md([)#])|\](./github.md\1|g" \
    -e "s|\]\(\.\./\.github/ARCHITECTURE\.md([)#])|\](./github.md\1|g" \
    "$full_src" > "$out"

  echo "  $src -> $out"
done

echo "Architecture docs copied to docs/architecture/"
