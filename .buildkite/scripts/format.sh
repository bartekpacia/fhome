#/usr/bin/env bash
set -euo pipefail

echo "~~~ Install formatter"
go install mvdan.cc/gofumpt@latest
echo "--- Verify formatting"
test -z "$(gofumpt -l .)"
