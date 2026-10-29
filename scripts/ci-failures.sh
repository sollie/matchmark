#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <run-id>" >&2
  exit 2
fi

gh run view "$1" --log-failed \
  | cut -f3- \
  | sed -E 's/^[0-9T:.Z-]+ (##\[error\])?//' \
  | grep -E -- '--- FAIL|^(FAIL|panic:)|\) \[[a-z]+\] ›|✘|Error:.*(expect|Timeout|locator)|^ *[0-9]+ (failed|flaky)' \
  | awk '!seen[$0]++' || echo "No failing test lines found in run $1" >&2
