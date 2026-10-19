# AGENTS.md — MatchMark

Agent instructions for working in this repository.

## Project Overview

MatchMark is a single-club WA1500/PPC competition scoring system shipped as one Go binary with an embedded Svelte SPA:
- `source/` — Go module root (`github.com/sollie/matchmark`)
  - `cmd/server/` — wiring, config, TLS modes, SPA serving
  - `internal/{handler,service,repository}` — layers; `internal/db` migrations; `internal/report` `.xlsx`/CSV; `internal/rulebook`
  - `web/` — Svelte + Vite SPA (admin, scorer, spectator), embedded from `web/dist`
  - `api/openapi.yaml` — API contract; generates `web/src/lib/api/schema.d.ts`
- `Dockerfile`, `docker-compose.yml` — single image, built on Alpine
- `scripts/smoke-image.sh` — in-image database smoke test (run by CI)

Specification and decisions:
- `docs/spec.md` — product spec
- `GLOSSARY.md` — vocabulary
- `docs/adr/` — architecture decision records
- `docs/v2-plan.md` — v2 rewrite steps

## General Rules

- Git write operations (branch, commit, push, tag, etc.) are allowed. Use Conventional
  Commits messages. Never force-push or rewrite history on `main`.
- When working through a list of items (e.g. a code review or security review), complete
  one item, run all required checks, and commit it before starting the next item.
- Never add comments to code. This is critical!
- Prefer the simplest solution that satisfies the requirement — avoid over-engineering.
- Before adding a new dependency, verify it is actively maintained and has recent releases. Prefer well-known, widely-used libraries over obscure alternatives.
- Write idiomatic code that follows the conventions and best practices of the language and ecosystem in use.
- Always verify that changes do not break existing functionality (run tests, type checks, linters as appropriate).
- All strings (UI labels, report labels, API field names, log messages) must be in **English**.
  No Swedish or other languages anywhere in the codebase.
- Do not commit generated or local artifacts (`codeql-db-*`, `results-*`, `.env`, `*.db`).

## Dependency Hygiene

- Always use the latest stable version of dependencies unless there is a documented reason not to.
- Before pinning any package or image version, check the upstream source (registry, GitHub releases, etc.) to confirm it is the current stable release.
- Do not leave stale, unused, or redundant dependencies in the project.

## Language-Specific Rules

### Go (`source/`)

- Use Go 1.27+. Run commands from `source/`.
- Architecture: `handler` → `service` → `repository`. No business logic in handlers or repositories.
- SQLite-format storage via `turso.tech/database/tursogo` (`database/sql` driver `turso`; `-tags musl` only for Alpine). `CGO_ENABLED=0` always. See ADR 0005.
- Schema evolves through ordered migrations in `internal/db/migrations/`; never edit an applied migration.
- HTTP router: `github.com/go-chi/chi/v5`. Reports via `excelize`. Passwords compared and hashed with `golang.org/x/crypto`.
- After every change, run:
  ```
  CGO_ENABLED=0 go build ./...
  CGO_ENABLED=0 go test ./...
  golangci-lint run ./...
  trivy fs --scanners vuln,secret .
  ```
- Fix all findings before considering the task done.

### TypeScript / Svelte (`source/web/`)

- Svelte 5 + Vite SPA, built to `web/dist` and embedded in the Go binary. No SSR, no service worker, no offline queue.
- The API client is generated from `source/api/openapi.yaml`. After changing the spec, run `npm run generate` in `source/api` and commit the result; `npm run check` there must leave no diff.
- After every change, run in `source/web`:
  ```
  npm run check
  npm run build
  npm audit
  ```
- Fix all high/critical findings reported by `npm audit` before considering the task done.

### Shell scripts

- Run `shellcheck` on any shell script before considering the task done.

## Security Constraints

- `ADMIN_PASSWORD` is required; the admin authenticates with a cookie session and login is rate limited.
- Scorers are paired devices, not accounts (ADR 0001). A device may only submit hits for its computed score lane; the lane is derived server-side from the pairing and current entries, never trusted from the client.
- Admins bypass lock checks and can unlock and overwrite any stage.
- All SQL queries use `?` placeholders via `database/sql` — no string concatenation (the only exception is `VACUUM INTO`, which needs a literal path).
- Client IPs come from forwarded headers only when the peer is in `TRUSTED_PROXY`.

## Rulebook Constants

Never hard-code shot counts or weapon class lists outside of `source/internal/rulebook/`.
That package is the single source of truth for:
- Course-of-fire definitions (stages, shot counts, distances, times)
- Allowed weapon classes per discipline
- 5-shot revolver variant detection (`Standard Revolver 2.75" 5-Shot` → 40-shot 48-course)

## Docker and Deployment

- The root `Dockerfile` is multi-stage: Node builds the SPA, Go builds the binary (`-tags musl`), final image is Alpine 3.24, non-root, data in `/data`.
- TLS is handled by the binary itself (plain, static certificate, or certmagic); there is no reverse-proxy service.
- Configuration is by environment variables, documented in `README.md`.

## Documentation

- Keep `README.md`, `docs/spec.md` and `GLOSSARY.md` accurate and up to date.
- Record significant decisions as ADRs in `docs/adr/`.
- After any change that affects how to run, build, or deploy the project locally or in
  production, update `README.md` in the same task.
- Tick completed steps in `docs/v2-plan.md` in the commit that completes them.
- Never leave the docs describing a state that no longer matches the codebase.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues (`sollie/matchmark`) via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-label vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
