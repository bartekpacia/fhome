#!/usr/bin/env bash
set -euo pipefail

echo "--- Run tests with coverage"
go test -cover ./...
