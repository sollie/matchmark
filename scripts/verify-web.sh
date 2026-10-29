#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

base=$(git merge-base HEAD origin/main 2>/dev/null || git rev-parse HEAD~1)
changed=$(git diff --name-only "$base" HEAD -- source/web source/api)

if [ -z "$changed" ]; then
  exit 0
fi

cd source/web
npm run verify

if echo "$changed" | grep -q '^source/web/src/' && [ -z "${SKIP_E2E:-}" ]; then
  npm run e2e
fi
