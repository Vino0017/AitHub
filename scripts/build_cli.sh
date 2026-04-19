#!/usr/bin/env bash
# Build aithub CLI for multiple platforms

set -euo pipefail

VERSION="3.0.0"
OUTPUT_DIR="dist"

echo "Building aithub CLI v${VERSION}..."

mkdir -p "$OUTPUT_DIR"

# Build for different platforms
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
  IFS='/' read -r os arch <<< "$platform"
  output_name="aithub-${os}-${arch}"

  if [ "$os" = "windows" ]; then
    output_name="${output_name}.exe"
  fi

  echo "  Building for ${os}/${arch}..."
  GOOS=$os GOARCH=$arch go build -ldflags="-s -w" -o "${OUTPUT_DIR}/${output_name}" ./cmd/aithub
done

echo "✓ Build complete. Binaries in ${OUTPUT_DIR}/"
ls -lh "$OUTPUT_DIR"
