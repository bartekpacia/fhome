#!/usr/bin/env bash
set -euo pipefail

echo "~~~ Fetch whole git history"
if [ -f .git/shallow ]; then
  git fetch --unshallow --tags
else
  git fetch --tags
fi

echo "+++ Run GoReleaser"
op run -- goreleaser release --clean
