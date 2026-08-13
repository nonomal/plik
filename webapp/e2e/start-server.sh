#!/bin/bash
# Start a fresh plikd instance for e2e testing.
# Creates a temp directory with clean SQLite DB + data dir, seeds admin user,
# then exec's plikd so Playwright can manage the process lifecycle.
set -e

TMPDIR=$(mktemp -d /tmp/plik-e2e.XXXXXX)
echo "$TMPDIR" > /tmp/plik-e2e-tmpdir  # so global-teardown.js can find it

WEBAPP_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SERVER_BIN="${PLIKD_BIN:-$WEBAPP_DIR/../server/plikd}"
PORT="${PLIK_PORT:-8585}"

if [[ ! -x "$SERVER_BIN" ]]; then
  echo "Error: plikd binary not found at $SERVER_BIN"
  echo "Run 'make server' first, or set PLIKD_BIN to the path of your plikd binary."
  exit 1
fi

mkdir -p "$TMPDIR/files"

# Write minimal config for e2e testing
cat > "$TMPDIR/plikd.cfg" << EOF
ListenPort            = $PORT
ListenAddress         = "0.0.0.0"
DataBackend           = "file"
FeatureAuthentication = "enabled"
FeatureLocalLogin     = "enabled"
WebappDirectory       = "$WEBAPP_DIR/dist"

[DataBackendConfig]
    Directory = "$TMPDIR/files"

[MetadataBackendConfig]
    Driver           = "sqlite3"
    ConnectionString = "$TMPDIR/plik.db"
EOF

# Seed admin user (runs against DB only, no HTTP server needed)
"$SERVER_BIN" --config "$TMPDIR/plikd.cfg" user create \
  --login admin --password plikplik --admin

# Start server (exec replaces shell so Playwright can manage the PID)
exec "$SERVER_BIN" --config "$TMPDIR/plikd.cfg"
