#!/bin/sh

exec 2>&1
set -e

exec caddy run \
  --config /bootstrap/Caddyfile \
  --adapter caddyfile
