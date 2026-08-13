#!/usr/bin/env bash

set -e

unamestr=$(uname)
if [ "$unamestr" = 'FreeBSD' ]; then
  MAKE="gmake"
else
  MAKE="make"
fi

RELEASE_VERSION=$(server/gen_build_info.sh version)

# Default client targets
if [[ -z "$CLIENT_TARGETS" ]]; then
  CLIENT_TARGETS="darwin/amd64,darwin/arm64,freebsd/386,freebsd/amd64,linux/386,linux/amd64,linux/arm,linux/arm64,openbsd/386,openbsd/amd64,windows/amd64,windows/386"
fi

echo ""
echo "Building clients for version $RELEASE_VERSION"
echo ""

rm -rf clients || true
mkdir -p clients/bash
cp client/plik.sh clients/bash

for TARGET in $(echo "$CLIENT_TARGETS" | awk -F, '{for (i=1;i<=NF;i++)print $i}')
do
  GOOS=$(echo "$TARGET" | cut -d "/" -f 1);
  export GOOS
	GOARCH=$(echo "$TARGET" | cut -d "/" -f 2);
	export GOARCH

  CLIENT_DIR="clients/${TARGET//\//-}"
  CLIENT_MD5="$CLIENT_DIR/MD5SUM"

  if [[ "$GOOS" == "windows" ]] ; then
    CLIENT_PATH="$CLIENT_DIR/plik.exe"
  else
    CLIENT_PATH="$CLIENT_DIR/plik"
  fi

  echo "################################################"
  echo "Building Plik client for $TARGET to $CLIENT_PATH"
  $MAKE --no-print-directory client

  mkdir -p "$CLIENT_DIR"
  cp client/plik "$CLIENT_PATH"
  md5sum "$CLIENT_PATH" | awk '{print $1}' > "$CLIENT_MD5"
done
