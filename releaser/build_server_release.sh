#!/usr/bin/env bash

set -e

unamestr=$(uname)
if [ "$unamestr" = 'FreeBSD' ]; then
  MAKE="gmake"
  TAR="gtar"
else
  MAKE="make"
  TAR="tar"
fi

# Assert frontend has been built already ( copied from previous docker stage )
if [[ ! -d "webapp/dist" ]]; then
  echo "Missing webapp distribution build"
  exit 1
fi

# Assert clients have been built already ( copied from previous docker stage )
if [[ ! -d "clients" ]]; then
  echo "Missing clients build"
  exit 1
fi

# Clean build artifacts but preserve pre-built clients and webapp
rm -rf server/plikd
rm -rf client/plik
rm -rf servers
rm -rf release
rm -rf releases

RELEASE_VERSION=$(server/gen_build_info.sh version)

echo ""
echo "Building Plik server v$RELEASE_VERSION $TARGETOS/$TARGETARCH$TARGETVARIANT"
echo ""

export GOOS=$TARGETOS
export GOARCH=$TARGETARCH
export GOARM=${TARGETVARIANT//v/}
export CGO_ENABLED=1

# set cross compiler
if [[ -z "$CC" ]]; then
  case "$TARGETARCH" in
    "amd64")
      unset CC
      ;;
    "386")
      export CC=i686-linux-gnu-gcc
      ;;
    "arm")
      export CC=arm-linux-gnueabi-gcc
      ;;
    "arm64")
      export CC=aarch64-linux-gnu-gcc
      ;;
  esac
fi

$MAKE --no-print-directory server

echo ""
echo "Building Plik release v$RELEASE_VERSION $TARGETOS/$TARGETARCH$TARGETVARIANT"
echo ""

RELEASE_DIR="release"

mkdir $RELEASE_DIR
mkdir $RELEASE_DIR/webapp
mkdir $RELEASE_DIR/server

# Copy release artifacts
cp -r clients $RELEASE_DIR
cp -r changelog $RELEASE_DIR
cp -r webapp/dist $RELEASE_DIR/webapp/dist
cp server/plikd.cfg $RELEASE_DIR/server
cp server/plikd $RELEASE_DIR/server/plikd

RELEASE="plik-server-$RELEASE_VERSION-$GOOS-$GOARCH"
RELEASE_ARCHIVE="$RELEASE.tar.gz"

echo ""
echo "Building Plik release archive $RELEASE_ARCHIVE"
echo ""

$TAR czf $RELEASE_ARCHIVE --transform "s,^$RELEASE_DIR,$RELEASE," $RELEASE_DIR
$TAR tf $RELEASE_ARCHIVE