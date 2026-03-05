#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v codeql > /dev/null 2>&1; then
    echo "codeql not found. Install it with: brew install codeql" >&2
    exit 1
fi

cd "$REPO_ROOT"

echo "==> Downloading CodeQL query packs (skipped if already cached)..."
codeql pack download codeql/go-queries codeql/javascript-queries

echo "==> Creating Go CodeQL database..."
CGO_ENABLED=0 codeql database create codeql-db-go \
    --language=go \
    --source-root=backend \
    --overwrite

echo "==> Analyzing Go database (security-and-quality)..."
codeql database analyze codeql-db-go \
    --format=sarif-latest \
    --output=results-go.sarif \
    codeql/go-queries:codeql-suites/go-security-and-quality.qls
codeql database analyze codeql-db-go \
    --format=csv \
    --output=results-go.csv \
    codeql/go-queries:codeql-suites/go-security-and-quality.qls

echo "==> Creating JavaScript/TypeScript CodeQL database..."
codeql database create codeql-db-js \
    --language=javascript-typescript \
    --source-root=frontend \
    --build-mode=none \
    --overwrite

echo "==> Analyzing JavaScript/TypeScript database (security-and-quality)..."
codeql database analyze codeql-db-js \
    --format=sarif-latest \
    --output=results-js.sarif \
    codeql/javascript-queries:codeql-suites/javascript-security-and-quality.qls
codeql database analyze codeql-db-js \
    --format=csv \
    --output=results-js.csv \
    codeql/javascript-queries:codeql-suites/javascript-security-and-quality.qls

echo ""

python3 - <<'EOF'
import csv
import sys

COLS = ["name", "description", "severity", "message", "path", "start_line", "start_col", "end_line", "end_col"]

def load(path):
    try:
        with open(path, newline="") as f:
            return list(csv.reader(f))
    except FileNotFoundError:
        return []

def print_section(label, rows):
    print(f"{label}")
    if not rows:
        print("  No findings.")
        print()
        return
    severity_w = max(len(r[2]) for r in rows)
    name_w     = max(len(r[0]) for r in rows)
    path_w     = max(len(r[4]) for r in rows)
    hdr = f"  {'SEVERITY':<{severity_w}}  {'RULE':<{name_w}}  {'FILE':<{path_w}}  {'LINE':>5}  MESSAGE"
    print(hdr)
    print("  " + "-" * (len(hdr) - 2))
    for r in rows:
        name, _desc, severity, message, path, start_line = r[0], r[1], r[2], r[3], r[4], r[5]
        print(f"  {severity:<{severity_w}}  {name:<{name_w}}  {path:<{path_w}}  {start_line:>5}  {message}")
    print()

go_rows = load("results-go.csv")
js_rows = load("results-js.csv")

print("=" * 60)
print("CodeQL Results")
print("=" * 60)
print()
print_section("Go (backend/)", go_rows)
print_section("JavaScript/TypeScript (frontend/)", js_rows)

total = len(go_rows) + len(js_rows)
print(f"Total findings: {total}")
if total > 0:
    print()
    print("Full details:")
    print("  results-go.sarif / results-go.csv")
    print("  results-js.sarif / results-js.csv")
    sys.exit(1)
EOF
