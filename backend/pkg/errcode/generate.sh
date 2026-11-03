#!/usr/bin/env bash

set -euo pipefail

backend_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "$backend_dir"
go run github.com/fino-io/finokit/errorx/cmd/errorxgen \
  -out ./pkg/errcode \
  -doc-out ./pkg/errcode/errcodes.md \
  -pkg errcode \
  ./pkg/errcode/configs
