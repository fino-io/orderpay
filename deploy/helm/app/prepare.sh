#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob

CHART_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$CHART_DIR/../../.." && pwd)"
if [[ $# -ne 1 || ! "$1" =~ ^[a-z0-9][a-z0-9-]*-server$ ]]; then
  echo 'Usage: bash prepare.sh <service>-server' >&2; exit 2
fi
SERVICE="$1"
CONFIG_DIR="$PROJECT_ROOT/backend/configs/$SERVICE"
CONFIG_FILES=("$CONFIG_DIR"/*.yaml)
if [[ ! -f "$CHART_DIR/services/$SERVICE.yaml" || ! -f "$CONFIG_DIR/server.yaml" || ${#CONFIG_FILES[@]} -eq 0 ]]; then
  echo "Missing service values or runtime YAML: $SERVICE" >&2; exit 2
fi

# Helm .Files can only read files inside the Chart. This cache is disposable.
umask 077
CACHE_DIR="$PROJECT_ROOT/.fino/cache/helm"
mkdir -p "$CACHE_DIR"
WORK_DIR="$(mktemp -d "$CACHE_DIR/.prepare.XXXXXX")"
trap 'rm -rf "$WORK_DIR"' EXIT
mkdir -p "$WORK_DIR/chart/configs"
cp "$CHART_DIR/Chart.yaml" "$CHART_DIR/values.yaml" "$CHART_DIR/values.schema.json" "$WORK_DIR/chart/"
cp -R "$CHART_DIR/templates" "$WORK_DIR/chart/"
cp "${CONFIG_FILES[@]}" "$WORK_DIR/chart/configs/"
rm -rf "$CACHE_DIR/$SERVICE"
mv "$WORK_DIR/chart" "$CACHE_DIR/$SERVICE"
printf '%s\n' "$CACHE_DIR/$SERVICE"
