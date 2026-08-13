#!/usr/bin/env bash

set -e

# Only inject version in CI to avoid modifying source files locally
if [[ -z "$CI" ]]; then
    echo "Skipping version injection (not in CI)"
    exit 0
fi

# Get the version from gen_build_info.sh
VERSION=$(../server/gen_build_info.sh version)

if [[ -z "$VERSION" ]]; then
    echo "Error: Could not determine version"
    exit 1
fi

echo "Injecting version $VERSION into documentation..."

# Replace __VERSION__ placeholders in all markdown files
find . -name '*.md' -not -path './node_modules/*' -not -path './vendor/*' -exec \
    sed -i "s/__VERSION__/$VERSION/g" {} +

echo "Version injection complete"
