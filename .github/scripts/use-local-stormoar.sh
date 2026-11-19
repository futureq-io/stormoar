#!/usr/bin/env bash
set -euo pipefail

stormoar_root=$1
upstream_module=github.com/lni/dragonboat/v4
local_module=github.com/futureq-io/stormoar/v4

# The pinned upstream consumer still imports Dragonboat. Use the Stormoar
# module path so its public types and internal imports resolve consistently.
git grep -lz -F "$upstream_module" -- '*.go' |
  xargs -0 perl -pi -e 's{github\.com/lni/dragonboat/v4}{github.com/futureq-io/stormoar/v4}g'
go mod edit -droprequire="$upstream_module" -require="$local_module@v4.0.0" \
  -replace="$local_module=$stormoar_root"
go mod tidy
