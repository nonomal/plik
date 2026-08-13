#!/bin/bash

#
## plik.sh test suite
##
## Usage:
##   bash client/plik_bash_test.sh              # Run all tests (unit + integration, needs server on :8080)
##   bash client/plik_bash_test.sh --unit        # Run unit tests only (no server needed)
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLIK_SH="$SCRIPT_DIR/plik.sh"
PLIK_URL="${PLIK_URL:-http://127.0.0.1:8080}"

PASS=0
FAIL=0
SKIP=0

green='\e[0;32m'
red='\e[0;31m'
yellow='\e[0;33m'
endColor='\e[0m'

function assert_eq() {
    local desc="$1" expected="$2" actual="$3"
    if [ "$expected" == "$actual" ]; then
        echo -e "  ${green}✓${endColor} $desc"
        PASS=$((PASS+1))
    else
        echo -e "  ${red}✗${endColor} $desc"
        echo "    expected: $expected"
        echo "    actual:   $actual"
        FAIL=$((FAIL+1))
    fi
}

function assert_contains() {
    local desc="$1" needle="$2" haystack="$3"
    if echo "$haystack" | grep -q "$needle"; then
        echo -e "  ${green}✓${endColor} $desc"
        PASS=$((PASS+1))
    else
        echo -e "  ${red}✗${endColor} $desc"
        echo "    expected to contain: $needle"
        echo "    actual: $haystack"
        FAIL=$((FAIL+1))
    fi
}

function assert_not_empty() {
    local desc="$1" value="$2"
    if [ "$value" != "" ]; then
        echo -e "  ${green}✓${endColor} $desc"
        PASS=$((PASS+1))
    else
        echo -e "  ${red}✗${endColor} $desc"
        echo "    expected non-empty value"
        FAIL=$((FAIL+1))
    fi
}

function skip_test() {
    local desc="$1"
    echo -e "  ${yellow}⊘${endColor} $desc (skipped)"
    SKIP=$((SKIP+1))
}

# Source helper functions from plik.sh without executing the main script
eval "$(sed -n '/^function /,/^}/p' "$PLIK_SH")"

# ============================================================================
# Unit Tests (no server needed)
# ============================================================================

echo ""
echo "=== Unit Tests ==="
echo ""

# -- urlencode --
echo "urlencode:"
assert_eq "encodes spaces" \
    "http://example.com/file/abc/my%20file.txt" \
    "$(urlencode "http://example.com/file/abc/my file.txt")"

assert_eq "encodes parentheses" \
    "http://example.com/file%20%281%29.txt" \
    "$(urlencode "http://example.com/file (1).txt")"

assert_eq "encodes UTF-8 (café)" \
    "http://example.com/caf%C3%A9.txt" \
    "$(urlencode "http://example.com/café.txt")"

assert_eq "preserves safe chars" \
    "http://example.com/normal-file_v1.0.txt" \
    "$(urlencode "http://example.com/normal-file_v1.0.txt")"

assert_eq "encodes ampersand" \
    "http://example.com/a%26b.txt" \
    "$(urlencode "http://example.com/a&b.txt")"

assert_eq "encodes hash" \
    "http://example.com/a%23b.txt" \
    "$(urlencode "http://example.com/a#b.txt")"

assert_eq "encodes percent" \
    "http://example.com/100%25.txt" \
    "$(urlencode "http://example.com/100%.txt")"

assert_eq "handles empty string" "" "$(urlencode "")"

# -- setTtl --
echo ""
echo "setTtl:"
TTL=0; setTtl "30m";  assert_eq "30m = 1800s"   "1800"   "$TTL"
TTL=0; setTtl "2h";   assert_eq "2h = 7200s"    "7200"   "$TTL"
TTL=0; setTtl "7d";   assert_eq "7d = 604800s"  "604800" "$TTL"
TTL=0; setTtl "3600"; assert_eq "3600 = 3600s"  "3600"   "$TTL"
TTL=0; setTtl "1d";   assert_eq "1d = 86400s"   "86400"  "$TTL"

# -- jsonValue --
echo ""
echo "jsonValue:"
UPLOAD_JSON='{"id":"abc123","uploadToken":"tok456","downloadDomain":"http://dl.example.com","oneShot":false}'
FILE_JSON='{"id":"file789","fileName":"test.txt","fileMd5":"deadbeef","status":"uploaded","fileSize":42}'

assert_eq "extracts id from upload" \
    "abc123" \
    "$(echo "$UPLOAD_JSON" | jsonValue id)"

assert_eq "extracts uploadToken" \
    "tok456" \
    "$(echo "$UPLOAD_JSON" | jsonValue uploadToken)"

assert_eq "extracts downloadDomain" \
    "http://dl.example.com" \
    "$(echo "$UPLOAD_JSON" | jsonValue downloadDomain)"

assert_eq "extracts id from file" \
    "file789" \
    "$(echo "$FILE_JSON" | jsonValue id)"

assert_eq "extracts fileName" \
    "test.txt" \
    "$(echo "$FILE_JSON" | jsonValue fileName)"

# -- help and version --
echo ""
echo "CLI flags:"
assert_contains "help flag" "Usage:" "$(bash "$PLIK_SH" -h 2>&1)"
assert_contains "help long flag" "Usage:" "$(bash "$PLIK_SH" --help 2>&1)"

# -- error handling --
echo ""
echo "Error handling:"
ERR_OUTPUT=$(bash "$PLIK_SH" 2>&1 || true)
assert_contains "no files error to stderr" "No files specified" "$ERR_OUTPUT"

BAD_OPT=$(bash "$PLIK_SH" --bogus 2>&1 || true)
assert_contains "bad option error" "bad option" "$BAD_OPT"

# ============================================================================
# Integration Tests (need a running server)
# ============================================================================

if [ "${1:-}" == "--unit" ]; then
    echo ""
    echo "=== Skipping integration tests (--unit mode) ==="
    echo ""
    echo -e "Results: ${green}$PASS passed${endColor}, ${red}$FAIL failed${endColor}, ${yellow}$SKIP skipped${endColor}"
    [ "$FAIL" -gt 0 ] && exit 1
    exit 0
fi

# Check server is reachable
if ! curl -sf "$PLIK_URL/version" > /dev/null 2>&1; then
    echo ""
    echo "=== Skipping integration tests (server not reachable at $PLIK_URL) ==="
    echo ""
    echo -e "Results: ${green}$PASS passed${endColor}, ${red}$FAIL failed${endColor}, ${yellow}$SKIP skipped${endColor}"
    [ "$FAIL" -gt 0 ] && exit 1
    exit 0
fi

echo ""
echo "=== Integration Tests (server: $PLIK_URL) ==="
echo ""

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

# -- version flag (needs server) --
echo "Version:"
VER_OUTPUT=$(bash "$PLIK_SH" -u "$PLIK_URL" -v 2>&1)
assert_contains "version flag returns server version" "version" "$VER_OUTPUT"

# -- basic upload and download --
echo "Upload and download:"
echo "hello world" > "$TMPDIR/test.txt"
OUTPUT=$(bash "$PLIK_SH" -u "$PLIK_URL" "$TMPDIR/test.txt")

# Extract the curl command and strip the > redirect so output goes to stdout
CURL_CMD=$(echo "$OUTPUT" | grep "^curl")
assert_not_empty "output contains curl command" "$CURL_CMD"

DL_CMD=$(echo "$CURL_CMD" | sed "s/ > .*$//")
DOWNLOADED=$(eval "$DL_CMD" 2>/dev/null || echo "DOWNLOAD_FAILED")
assert_eq "downloaded content matches" "hello world" "$DOWNLOADED"

# -- file with spaces --
echo ""
echo "File with spaces:"
echo "spaced content" > "$TMPDIR/file with spaces.txt"
SPACE_OUTPUT=$(bash "$PLIK_SH" -u "$PLIK_URL" "$TMPDIR/file with spaces.txt")
SPACE_CMD=$(echo "$SPACE_OUTPUT" | grep "^curl")

assert_contains "URL is percent-encoded" "%20" "$SPACE_CMD"
assert_contains "filename is quoted" "'file with spaces.txt'" "$SPACE_CMD"

SPACE_DL_CMD=$(echo "$SPACE_CMD" | sed "s/ > .*$//")
SPACE_DL=$(eval "$SPACE_DL_CMD" 2>/dev/null || echo "DOWNLOAD_FAILED")
assert_eq "download with spaces works" "spaced content" "$SPACE_DL"

# -- UTF-8 filename --
echo ""
echo "UTF-8 filename:"
echo "accented" > "$TMPDIR/café.txt"
UTF8_OUTPUT=$(bash "$PLIK_SH" -u "$PLIK_URL" "$TMPDIR/café.txt")
UTF8_CMD=$(echo "$UTF8_OUTPUT" | grep "^curl")

assert_contains "UTF-8 encoded in URL" "%C3%A9" "$UTF8_CMD"

UTF8_DL_CMD=$(echo "$UTF8_CMD" | sed "s/ > .*$//")
UTF8_DL=$(eval "$UTF8_DL_CMD" 2>/dev/null || echo "DOWNLOAD_FAILED")
assert_eq "download UTF-8 file works" "accented" "$UTF8_DL"

# -- quiet mode --
echo ""
echo "Quiet mode:"
echo "quiet test" > "$TMPDIR/quiet.txt"
QUIET_OUTPUT=$(bash "$PLIK_SH" -u "$PLIK_URL" -q "$TMPDIR/quiet.txt")

assert_not_empty "quiet mode outputs URL" "$QUIET_OUTPUT"
# In quiet mode output should be just the URL, no curl prefix
if echo "$QUIET_OUTPUT" | grep -q "^curl"; then
    echo -e "  ${red}✗${endColor} quiet mode should not output curl command"
    FAIL=$((FAIL+1))
else
    echo -e "  ${green}✓${endColor} quiet mode outputs URL only"
    PASS=$((PASS+1))
fi

# Verify the quiet-mode URL is downloadable
QUIET_DL=$(curl -s "$QUIET_OUTPUT" 2>/dev/null || echo "DOWNLOAD_FAILED")
assert_eq "quiet mode URL is downloadable" "quiet test" "$QUIET_DL"

# -- oneshot --
echo ""
echo "OneShot mode:"
echo "oneshot content" > "$TMPDIR/oneshot.txt"
OS_OUTPUT=$(bash "$PLIK_SH" -u "$PLIK_URL" -o "$TMPDIR/oneshot.txt")
OS_CMD=$(echo "$OS_OUTPUT" | grep "^curl")

OS_DL_CMD=$(echo "$OS_CMD" | sed "s/ > .*$//")
OS_DL1=$(eval "$OS_DL_CMD" 2>/dev/null || echo "DOWNLOAD_FAILED")
assert_eq "first download succeeds" "oneshot content" "$OS_DL1"

OS_DL2=$(eval "$OS_DL_CMD" 2>/dev/null)
OS_STATUS=$?
# Second download should fail (file deleted)
if echo "$OS_DL2" | grep -qi "not found\|deleted\|error\|gone"; then
    echo -e "  ${green}✓${endColor} second download fails (file deleted)"
    PASS=$((PASS+1))
elif [ "$OS_DL2" == "oneshot content" ]; then
    echo -e "  ${red}✗${endColor} second download should have failed but got content"
    FAIL=$((FAIL+1))
else
    echo -e "  ${green}✓${endColor} second download fails (file deleted)"
    PASS=$((PASS+1))
fi

# -- error on bad server --
echo ""
echo "Error handling (bad server):"
set +e
bash "$PLIK_SH" -u "http://127.0.0.1:1" "$TMPDIR/test.txt" > /dev/null 2>&1
BAD_EXIT=$?
set -euo pipefail
if [ "$BAD_EXIT" != "0" ]; then
    echo -e "  ${green}✓${endColor} bad server produces non-zero exit"
    PASS=$((PASS+1))
else
    echo -e "  ${red}✗${endColor} bad server should produce non-zero exit"
    FAIL=$((FAIL+1))
fi

# ============================================================================
# Summary
# ============================================================================

echo ""
echo "============================================"
echo -e "Results: ${green}$PASS passed${endColor}, ${red}$FAIL failed${endColor}, ${yellow}$SKIP skipped${endColor}"
echo "============================================"

[ "$FAIL" -gt 0 ] && exit 1
exit 0
