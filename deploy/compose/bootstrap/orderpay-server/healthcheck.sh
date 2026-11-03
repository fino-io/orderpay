#!/bin/sh

set -e

if wget \
    -q \
    --spider \
    "http://127.0.0.1:${SERVER_DEBUG_PORT:?Set SERVER_DEBUG_PORT}/readyz" \
    2>/dev/null; then
  exit 0
else
  exit 1
fi
