#/usr/bin/env bash
set -euo pipefail

echo "~~~ Install linter"
go install honnef.co/go/tools/cmd/staticcheck@latest
echo "--- Verify linting"
staticcheck ./...
