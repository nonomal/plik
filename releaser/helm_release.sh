#!/usr/bin/env bash

set -e

#
# Package the Helm chart and update the gh-pages Helm repo index.
#
# Usage:
#   releaser/helm_release.sh <version>
#
# Environment:
#   DRY_RUN=true  — package chart and generate index.yaml without pushing to gh-pages
#
# This script:
#   1. Replaces __VERSION__ placeholders in Chart.yaml with the release tag
#   2. Packages the chart as plik-helm-<version>.tgz into releases/
#   3. Fetches the existing index.yaml from gh-pages
#   4. Merges the new chart entry via `helm repo index --merge`
#   5. Commits the updated index.yaml to gh-pages (skipped in dry-run mode)
#
# The packaged .tgz is left in releases/ so the caller can upload it
# as a GitHub release artifact alongside other release files.
#

CHART_YAML="charts/plik/Chart.yaml"

TAG=$1
if [[ -z "$TAG" ]]; then
  echo "Usage: $0 <version>"
  exit 1
fi

GIT_REMOTE=${GIT_REMOTE:-origin}

echo ""
echo " Packaging Helm chart for $TAG"
echo ""

# Replace __VERSION__ placeholders with the release tag
sed -i "s/^version: __VERSION__/version: $TAG/" "$CHART_YAML"
sed -i "s/^appVersion: \"__VERSION__\"/appVersion: \"$TAG\"/" "$CHART_YAML"

# Package chart into releases/
mkdir -p releases
helm package charts/plik -d releases/

# Rename to plik-helm-<version>.tgz
# helm package produces plik-$TAG.tgz (from Chart.yaml version we just set)
mv "releases/plik-${TAG}.tgz" "releases/plik-helm-${TAG}.tgz"

echo ""
echo " Updating Helm repo index on gh-pages"
echo ""

# Fetch gh-pages branch
git fetch "$GIT_REMOTE" gh-pages

# Prepare a temp directory with the existing index and new chart
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

git show "$GIT_REMOTE"/gh-pages:index.yaml > "$TMPDIR/index.yaml" 2>/dev/null || true
cp "releases/plik-helm-${TAG}.tgz" "$TMPDIR/"

# Merge new chart entry into the index
# The --url points to the GitHub release download URL for this tag
helm repo index "$TMPDIR" \
  --url "https://github.com/root-gg/plik/releases/download/${TAG}" \
  --merge "$TMPDIR/index.yaml"

if [[ "$DRY_RUN" == "true" ]]; then
  echo ""
  echo " [DRY RUN] Skipping gh-pages push. Generated index.yaml:"
  echo ""
  cat "$TMPDIR/index.yaml"
  echo ""
  echo " [DRY RUN] Chart artifact: releases/plik-helm-${TAG}.tgz"
  ls -l "releases/plik-helm-${TAG}.tgz"
  echo ""
  echo " [DRY RUN] Reverting Chart.yaml"
  sed -i "s/^version: $TAG/version: __VERSION__/" "$CHART_YAML"
  sed -i "s/^appVersion: \"$TAG\"/appVersion: \"__VERSION__\"/" "$CHART_YAML"
  exit 0
fi

# Commit updated index.yaml to gh-pages using a detached worktree
# to avoid disturbing the current working directory
WORKTREE=$(mktemp -d)
trap "rm -rf $TMPDIR $WORKTREE" EXIT

git worktree add --detach "$WORKTREE" "$GIT_REMOTE/gh-pages"
cp "$TMPDIR/index.yaml" "$WORKTREE/index.yaml"

cd "$WORKTREE"
git add index.yaml
git commit -m "Update Helm repo index for $TAG"
git push "$GIT_REMOTE" HEAD:gh-pages
cd -

git worktree remove "$WORKTREE"

echo ""
echo " Helm chart plik-helm-${TAG}.tgz packaged and index.yaml updated"
echo ""
