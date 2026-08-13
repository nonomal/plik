#!/usr/bin/env bash

set -e

#
# Build .deb packages and update the APT repository on gh-pages.
#
# Usage:
#   releaser/apt_release.sh <version>
#
# Environment:
#   DRY_RUN=true      — build packages and generate repo metadata without pushing to gh-pages
#   SKIP_BUILD=true   — skip nfpm packaging, use existing .deb files in releases/
#   GPG_PRIVATE_KEY   — armored GPG private key for signing the APT repository
#   GIT_REMOTE        — git remote to push gh-pages to (default: origin)
#
# This script:
#   1. Builds .deb packages for plik-server and plik-client using nfpm
#   2. Fetches the existing APT repo from gh-pages
#   3. Adds the new packages to the pool
#   4. Generates Packages, Release, and InRelease metadata
#   5. Signs the repo with GPG
#   6. Commits and pushes to gh-pages (skipped in dry-run mode)
#
# The .deb files are left in releases/ so the caller can upload them
# as GitHub release artifacts alongside other release files.
#

TAG=$1
if [[ -z "$TAG" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

GIT_REMOTE=${GIT_REMOTE:-origin}

# Map Go architecture names to Debian architecture names
declare -A ARCH_MAP=(
  ["amd64"]="amd64"
  ["386"]="i386"
  ["arm64"]="arm64"
  ["arm"]="armhf"
)

# Ensure releases directory exists
mkdir -p releases

if [[ "$SKIP_BUILD" == "true" ]]; then
  echo ""
  echo " Skipping .deb build, using existing packages in releases/"
  echo ""
else
  echo ""
  echo " Building Debian packages for $TAG"
  echo ""

  # Build .deb packages from the release archives
  for RELEASE_ARCHIVE in releases/plik-server-*.tar.gz; do
    [[ -f "$RELEASE_ARCHIVE" ]] || continue

    # Extract Go arch from filename: plik-server-<version>-linux-<goarch>.tar.gz
    GOARCH=$(basename "$RELEASE_ARCHIVE" .tar.gz | sed "s/plik-server-${TAG}-linux-//")
    DEB_ARCH=${ARCH_MAP[$GOARCH]:-}
    if [[ -z "$DEB_ARCH" ]]; then
      echo "Error: unknown Go architecture '$GOARCH', please add it to ARCH_MAP"
      exit 1
    fi

    echo ""
    echo " Building plik-server $TAG ($DEB_ARCH) from $RELEASE_ARCHIVE"
    echo ""

    # Extract the release archive into the release/ directory expected by nfpm configs
    rm -rf release
    mkdir -p release
    tar xzf "$RELEASE_ARCHIVE" -C release --strip-components=1

    # Build server .deb
    VERSION="$TAG" DEB_ARCH="$DEB_ARCH" \
      nfpm pkg \
        --config releaser/nfpm-server.yaml \
        --packager deb \
        --target "releases/plik-server_${TAG}_${DEB_ARCH}.deb"

    # Build client .deb
    # Copy the matching client binary for this architecture to the fixed path
    CLIENT_SRC="release/clients/linux-${GOARCH}/plik"
    if [[ -f "$CLIENT_SRC" ]]; then
      mkdir -p release/client
      cp "$CLIENT_SRC" release/client/plik
      VERSION="$TAG" DEB_ARCH="$DEB_ARCH" \
        nfpm pkg \
          --config releaser/nfpm-client.yaml \
          --packager deb \
          --target "releases/plik-client_${TAG}_${DEB_ARCH}.deb"
    else
      echo " Warning: no client binary found for linux-${GOARCH}, skipping plik-client .deb"
    fi

    rm -rf release
  done
fi

# Verify that we have a plik-server .deb for every expected architecture
MISSING=()
for DEB_ARCH in "${ARCH_MAP[@]}"; do
  if ! ls releases/plik-server_*_${DEB_ARCH}.deb 1>/dev/null 2>&1; then
    MISSING+=("$DEB_ARCH")
  fi
done

if [[ ${#MISSING[@]} -gt 0 ]]; then
  echo "Error: missing plik-server .deb for architecture(s): ${MISSING[*]}"
  echo "Found packages:"
  ls -l releases/*.deb 2>/dev/null || echo "  (none)"
  exit 1
fi

DEB_COUNT=$(find releases -name "*.deb" | wc -l)
echo ""
echo " Found $DEB_COUNT .deb package(s):"
ls -l releases/*.deb
echo ""

# --- APT repository update ---

echo ""
echo " Updating APT repository on gh-pages"
echo ""

# Import GPG key if provided
if [[ -n "$GPG_PRIVATE_KEY" ]]; then
  GPG_IMPORT_OUTPUT=$(echo "$GPG_PRIVATE_KEY" | gpg --batch --import 2>&1)
  echo "$GPG_IMPORT_OUTPUT"
  GPG_KEY_ID=$(echo "$GPG_IMPORT_OUTPUT" | grep -oE 'key [0-9A-F]+' | head -1 | awk '{print $2}')
  echo " Using GPG key: $GPG_KEY_ID"

  # Configure gpg-agent for non-interactive signing in CI
  mkdir -p ~/.gnupg
  chmod 700 ~/.gnupg
  echo "allow-loopback-pinentry" > ~/.gnupg/gpg-agent.conf
  gpg-connect-agent reloadagent /bye 2>/dev/null || true
else
  if [[ "$DRY_RUN" != "true" ]]; then
    echo "Error: GPG_PRIVATE_KEY is required for signing (set DRY_RUN=true to skip)"
    exit 1
  fi
  echo " Warning: no GPG key provided, skipping signing"
fi

# Fetch gh-pages branch (may not exist yet on first run)
git fetch "$GIT_REMOTE" gh-pages 2>/dev/null || true

# Prepare a temp worktree
WORKTREE=$(mktemp -d)
trap "rm -rf $WORKTREE" EXIT

if git rev-parse --verify "$GIT_REMOTE/gh-pages" >/dev/null 2>&1; then
  # Clone existing gh-pages content so we keep previous packages in the pool
  git worktree add --detach "$WORKTREE" "$GIT_REMOTE/gh-pages"
  # Convert to an orphan branch (discard history, keep files)
  cd "$WORKTREE"
  git checkout --orphan gh-pages-orphan
  cd -
else
  # First run: create a fresh worktree with an orphan branch
  git worktree add --detach "$WORKTREE" HEAD
  cd "$WORKTREE"
  git checkout --orphan gh-pages-orphan
  git rm -rf . >/dev/null 2>&1 || true
  cd -
fi

# Create APT repo directory structure
APT_DIR="$WORKTREE/apt"
mkdir -p "$APT_DIR/pool/main"

# Remove old versions of packages being replaced
for DEB in releases/*.deb; do
  [[ -f "$DEB" ]] || continue
  PKG_NAME=$(dpkg-deb -f "$DEB" Package)
  DEB_ARCH=$(dpkg-deb -f "$DEB" Architecture)
  # Remove any existing versions of this package for this architecture
  find "$APT_DIR/pool/main/" -name "${PKG_NAME}_*_${DEB_ARCH}.deb" -delete 2>/dev/null || true
done

# Copy .deb files to the pool
cp releases/*.deb "$APT_DIR/pool/main/"

# Generate per-architecture Packages files
for DEB_ARCH in "${ARCH_MAP[@]}"; do
  ARCH_DIR="$APT_DIR/dists/stable/main/binary-${DEB_ARCH}"
  mkdir -p "$ARCH_DIR"

  # Generate Packages file for this architecture
  cd "$APT_DIR"
  dpkg-scanpackages --arch "$DEB_ARCH" pool/main > "dists/stable/main/binary-${DEB_ARCH}/Packages"
  gzip -k -f "dists/stable/main/binary-${DEB_ARCH}/Packages"
  cd -
done

# Generate Release file
cd "$APT_DIR/dists/stable"

ARCHITECTURES=$(echo "${ARCH_MAP[@]}" | tr ' ' '\n' | sort -u | tr '\n' ' ')

cat > Release <<EOF
Origin: root-gg
Label: Plik
Suite: stable
Codename: stable
Architectures: ${ARCHITECTURES}
Components: main
Description: Plik APT repository
Date: $(date -Ru)
EOF

# Add checksums to Release file
{
  echo "MD5Sum:"
  find main -name "Packages*" -exec sh -c 'echo " $(md5sum "{}" | cut -d" " -f1) $(wc -c < "{}") {}"' \;
  echo "SHA256:"
  find main -name "Packages*" -exec sh -c 'echo " $(sha256sum "{}" | cut -d" " -f1) $(wc -c < "{}") {}"' \;
} >> Release

cd -

# GPG sign the Release file
if [[ -n "$GPG_KEY_ID" ]]; then
  # Detached signature
  gpg --batch --yes --armor --pinentry-mode loopback --passphrase "$GPG_PASSPHRASE" \
    --default-key "$GPG_KEY_ID" \
    --detach-sign \
    --output "$APT_DIR/dists/stable/Release.gpg" \
    "$APT_DIR/dists/stable/Release"

  # Inline signature (InRelease)
  gpg --batch --yes --armor --pinentry-mode loopback --passphrase "$GPG_PASSPHRASE" \
    --default-key "$GPG_KEY_ID" \
    --clearsign \
    --output "$APT_DIR/dists/stable/InRelease" \
    "$APT_DIR/dists/stable/Release"

  # Export public key
  gpg --armor --export "$GPG_KEY_ID" > "$APT_DIR/gpg.key"
fi

if [[ "$DRY_RUN" == "true" ]]; then
  echo ""
  echo " [DRY RUN] Skipping gh-pages push. Generated APT repo:"
  echo ""
  find "$APT_DIR" -type f | sort
  echo ""
  echo " [DRY RUN] Release file:"
  cat "$APT_DIR/dists/stable/Release"

  git worktree remove "$WORKTREE" --force
  exit 0
fi

# Commit and force-push to gh-pages (single orphan commit, no history)
cd "$WORKTREE"
git add -A
git commit -m "Update APT repository for $TAG"
git push --force "$GIT_REMOTE" HEAD:gh-pages
cd -

git worktree remove "$WORKTREE" --force

# Remove .deb files from releases/ — they are now in the APT repository
rm -f releases/*.deb

echo ""
echo " APT repository updated for $TAG"
echo ""
echo " Users can install with:"
echo "   curl -fsSL https://root-gg.github.io/plik/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/plik.gpg"
echo "   echo 'deb [signed-by=/etc/apt/keyrings/plik.gpg] https://root-gg.github.io/plik/apt stable main' | sudo tee /etc/apt/sources.list.d/plik.list"
echo "   sudo apt update && sudo apt install plik-server"
echo ""
