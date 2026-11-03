#!/usr/bin/env bash
set -euo pipefail

PLATFORM="${PLATFORM:-linux/amd64}"
REGISTRY="${REGISTRY:-}"
IMAGE_TAG="${IMAGE_TAG:-v1.0.0}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SERVICE=orderpay-server
IMAGE="${REGISTRY:+$REGISTRY/}$SERVICE:$IMAGE_TAG"

IFS=/ read -r goos goarch <<< "$PLATFORM"
bash "$PROJECT_ROOT/backend/cmd/$SERVICE/build.sh" "$goos" "$goarch"

docker build --platform "$PLATFORM" -t "$IMAGE" \
  -f "$PROJECT_ROOT/backend/build/$SERVICE/Dockerfile" "$PROJECT_ROOT/.fino/backend"
if [[ -n "$REGISTRY" ]]; then
  docker push "$IMAGE"
fi
