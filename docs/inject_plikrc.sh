#!/usr/bin/env bash
#
# Inject client/.plikrc into docs/features/cli-client.md at build time.
#
# Uses paired markers in the source file:
#   <!-- BEGIN:PLIKRC -->
#   <!-- END:PLIKRC -->
#
# Everything between these markers is replaced with the file contents
# wrapped in a ```toml code fence. This makes the script fully
# idempotent — safe to run multiple times.
#
# This runs as part of `make docs`.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PLIKRC="$REPO_ROOT/client/.plikrc"
TARGET="$SCRIPT_DIR/features/cli-client.md"

if [[ ! -f "$PLIKRC" ]]; then
    echo "WARNING: client/.plikrc not found, skipping injection"
    exit 0
fi

if ! grep -q '<!-- BEGIN:PLIKRC -->' "$TARGET"; then
    echo "WARNING: <!-- BEGIN:PLIKRC --> marker not found in cli-client.md, skipping"
    exit 0
fi

# Use awk to replace everything between BEGIN/END markers
awk -v plikrc="$PLIKRC" '
    /<!-- BEGIN:PLIKRC -->/ {
        print
        print "```toml"
        while ((getline line < plikrc) > 0) print line
        close(plikrc)
        print "```"
        skip = 1
        next
    }
    /<!-- END:PLIKRC -->/ {
        skip = 0
        print
        next
    }
    !skip { print }
' "$TARGET" > "$TARGET.tmp" && mv "$TARGET.tmp" "$TARGET"

echo "  client/.plikrc -> cli-client.md (injected)"
