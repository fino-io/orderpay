#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
SERVICE="$(basename "$SCRIPT_DIR")"
cd "$PROJECT_ROOT/backend"

goos="${1:-$(go env GOOS)}"
case "$goos" in
linux | darwin | "") ;;
*) echo "illegal os: $goos" && exit 1 ;;
esac

goarch="${2:-$(go env GOARCH)}"
case "$goarch" in
amd64 | arm64 | "") ;;
*) echo "illegal arch: $goarch" && exit 1 ;;
esac

OUTPUT_DIR="$PROJECT_ROOT/.fino/backend/$goos/$goarch"
mkdir -p "$OUTPUT_DIR"

# 构建信息
git_hash=$(git rev-parse --short HEAD)
git_branch=$(git rev-parse --abbrev-ref HEAD)
build_time=$(date '+%Y-%m-%d %H:%M:%S %Z')

env GOOS="$goos" GOARCH="$goarch" \
  go build \
  -o "$OUTPUT_DIR/$SERVICE" \
  -buildvcs=false \
  -ldflags "-s -w -X 'main.gitHash=$git_hash' -X 'main.gitBranch=$git_branch' -X 'main.buildTime=$build_time'" \
  "./cmd/$SERVICE"
