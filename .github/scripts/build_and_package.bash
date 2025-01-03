#!/usr/bin/env bash

#!/bin/bash

set -e

# Ensure a tag is passed as an argument
if [ -z "$1" ]; then
  echo "Usage: $0 <version-tag>"
  exit 1
fi

VERSION_TAG=$1
OUTPUT_DIR="dist"
BINARY_NAME="app"

# Clean and create the output directory
rm -rf $OUTPUT_DIR
mkdir -p $OUTPUT_DIR

# Platforms to build
platforms=(
  "linux amd64"
  "linux arm64"
  "darwin arm64"
)

echo "Building binaries for version: $VERSION_TAG"

# Build binaries
for platform in "${platforms[@]}"; do
  OS=${platform%% *}
  ARCH=${platform##* }

  OUTPUT_FILE="${BINARY_NAME}-${OS}-${ARCH}"

  echo "Building for $OS/$ARCH..."
  CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build -ldflags="-s -w" -o $OUTPUT_DIR/$OUTPUT_FILE ./cmd

  # Package binaries
  echo "Packaging $OUTPUT_FILE..."
  tar -zcf $OUTPUT_DIR/${OUTPUT_FILE}-${VERSION_TAG}.tar.gz -C $OUTPUT_DIR $OUTPUT_FILE
  zip -j $OUTPUT_DIR/${OUTPUT_FILE}-${VERSION_TAG}.zip $OUTPUT_DIR/$OUTPUT_FILE

  # Generate checksums
  echo "Generating checksum for $OUTPUT_FILE..."
  sha256sum $OUTPUT_DIR/${OUTPUT_FILE}-${VERSION_TAG}.tar.gz > $OUTPUT_DIR/${OUTPUT_FILE}-${VERSION_TAG}.tar.gz.sha256
  sha256sum $OUTPUT_DIR/${OUTPUT_FILE}-${VERSION_TAG}.zip > $OUTPUT_DIR/${OUTPUT_FILE}-${VERSION_TAG}.zip.sha256

  # Clean up individual binary to save space
  rm $OUTPUT_DIR/$OUTPUT_FILE
done

echo "Build and packaging completed. Artifacts are in the $OUTPUT_DIR directory."
