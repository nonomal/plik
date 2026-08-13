#!/usr/bin/env bash
#
# Inject `plik --help` output into docs/features/cli-client.md at build time.
#
# Uses paired markers in the source file:
#   <!-- BEGIN:HELP -->
#   <!-- END:HELP -->
#
# Everything between these markers is replaced with the help output
# wrapped in a ``` code fence. This makes the script fully
# idempotent — safe to run multiple times.
#
# Requires the client binary to be built first (make client).
# This runs as part of `make docs`.

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PLIK="$REPO_ROOT/client/plik"
TARGET="$SCRIPT_DIR/features/cli-client.md"

if [[ ! -x "$PLIK" ]]; then
    echo "WARNING: client/plik binary not found, skipping help injection (run 'make client' first)"
    exit 0
fi

if ! grep -q '<!-- BEGIN:HELP -->' "$TARGET"; then
    echo "WARNING: <!-- BEGIN:HELP --> marker not found in cli-client.md, skipping"
    exit 0
fi

# Capture help output
HELP_OUTPUT=$("$PLIK" --help 2>&1)

# Use awk to replace everything between BEGIN/END markers
awk -v help="$HELP_OUTPUT" '
    /<!-- BEGIN:HELP -->/ {
        print
        print "```"
        print help
        print "```"
        skip = 1
        next
    }
    /<!-- END:HELP -->/ {
        skip = 0
        print
        next
    }
    !skip { print }
' "$TARGET" > "$TARGET.tmp" && mv "$TARGET.tmp" "$TARGET"

echo "  plik --help -> cli-client.md (injected)"
