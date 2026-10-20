#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

hits=$(
  {
    grep -rnE '^\s*(//|/\*)' --include=*.go source | grep -vE '//(go:|nolint)' || true
    grep -rnE '^\s*(//|/\*|<!--)' --include=*.ts --include=*.svelte source/web/src source/web/e2e source/web/playwright.config.ts source/web/vite.config.ts \
      | grep -vE 'schema\.d\.ts|/// <reference' || true
  }
)

if [ -n "$hits" ]; then
  echo "Comments are not allowed:" >&2
  echo "$hits" >&2
  exit 1
fi
