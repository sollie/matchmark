#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"

"$root/scripts/check-no-comments.sh"
cd "$root/source"
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
golangci-lint run ./...
cd "$root"
trivy fs --scanners vuln,secret .
