# Copilot Instructions — MatchMark

Single-club WA1500/PPC competition scoring system: one Go binary (`source/`) with an embedded Svelte SPA (`source/web/`). The spec is `docs/spec.md`, vocabulary is `GLOSSARY.md`, decisions are in `docs/adr/`; `AGENTS.md` holds the binding agent rules.

## Build, test, lint

Backend (run in `source/`, always `CGO_ENABLED=0`):
```sh
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test ./internal/service -run TestName -v   # single test
golangci-lint run ./...
trivy fs --scanners vuln,secret .                           # from repo root
ADMIN_PASSWORD=<16+ chars> go run ./cmd/server
```

Frontend (run in `source/web/`; tests use Vitest with Testing Library and a stubbed fetch):
```sh
npm run check && npm run build && npm test && npm audit   # required after every change
```

API client: after editing `source/api/openapi.yaml`, run `npm run generate` in `source/api` and commit `web/src/lib/api/schema.d.ts`.

Shell scripts: `shellcheck`. Full stack: `docker compose up --build`.

## Architecture

- **Backend layering is strict:** `handler` → `service` → `repository` (under `source/internal/`). No business logic in handlers or repositories. Wiring, config, TLS and request logging are in `cmd/server/`.
- **Storage:** SQLite format via `turso.tech/database/tursogo` (ADR 0005); schema evolves through ordered migrations in `internal/db/migrations/` (never edit an applied migration). Optional push-only replication.
- **`internal/rulebook/` is the single source of truth** for courses of fire and shot counts. Never hard-code them elsewhere.
- **Scoring model:** the score lane is the next manned lane after the scorer's own lane, wrapping, derived server-side from the device pairing and current entries. A competition with one manned lane requires admin scoring. Hits are seven ring counts per stage that must sum to the expected shots. A lock records the scorer; only the admin unlocks, edits, or locks as "Admin".
- **Auth:** admin password from `ADMIN_PASSWORD` and a cookie session with login rate limiting. Scorers are paired devices (QR link), not accounts. Spectator endpoints are unauthenticated and gated by per-competition visibility flags.
- **Reports:** `.xlsx` filled from the embedded template via excelize, plus a detailed CSV.
- **Logging:** JSON `log/slog` to stdout, one `request` line per HTTP request.
- **Docker:** one image built from the root `Dockerfile`; TLS is handled by the binary (plain, static, or certmagic).

## Conventions

- **Never add comments to code.**
- All strings (UI, reports, API fields, logs) in English only.
- Dates are ISO 8601 (`2026-10-19`) and times are 24-hour.
- All SQL uses `?` placeholders; no string concatenation.
- Tests are table/unit tests in `internal/service`, `internal/handler`, `internal/rulebook` and `cmd/server`; follow the existing style.
- Use the latest stable dependency versions; verify upstream before pinning; remove unused deps. Fix high/critical `npm audit` findings.
- Keep `README.md`, `docs/spec.md` and `GLOSSARY.md` in sync with any change to behaviour, API, Docker or run/deploy steps.
- Git write operations are allowed (Conventional Commits; never force-push or rewrite history on `main`). When working through a list of items (reviews), finish one, run checks, and commit it before starting the next.
- Issues live in GitHub Issues; see the `## Agent skills` section of `AGENTS.md` and `docs/agents/`.
