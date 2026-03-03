# AGENTS.md — MatchMark

Agent instructions for working in this repository.

## Project Overview

MatchMark is a WA1500/PPC competition scoring system consisting of:
- `frontend/` — SvelteKit PWA (TypeScript, `@vite-pwa/sveltekit`, adapter-static)
- `backend/` — Go REST API (`chi`, `modernc.org/sqlite`, JWT auth)
- `caddy/Caddyfile` — TLS termination + reverse proxy + static file server
- `docker-compose.yml` — orchestrates all three services

See `docs/plan/` for the full specification:
- `00-index.md` — plan file index
- `01-architecture.md` — component overview and repository layout
- `02-database.md` — SQLite schema and design notes
- `03-backend.md` — API contract, auth, export layout
- `04-frontend.md` — routes, stores, offline sync, UI spec
- `05-rulebook.md` — authoritative WA1500 course-of-fire and weapon class data
- `06-decisions.md` — technology and design decisions with rationale
- `07-phases.md` — implementation phases and checklist

## General Rules

- Never do git write operations (commit, push, tag, etc.)
- Never add comments to code.
- All strings (UI labels, PDF labels, API field names, log messages) must be in **English**.
  No Swedish or other languages anywhere in the codebase.

## Language-Specific Rules

### Go (`backend/`)

- Use Go 1.26+.
- Module path: `github.com/sollie/matchmark`.
- Architecture: `handler` → `service` → `repository`. No business logic in handlers or repositories.
- SQLite via `modernc.org/sqlite` (pure Go, `database/sql` compatible). `CGO_ENABLED=0` always.
- JWT via `github.com/golang-jwt/jwt/v5`. Passwords hashed with argon2id (`golang.org/x/crypto`).
- HTTP router: `github.com/go-chi/chi/v5`.
- After every change, run:
  ```
  golangci-lint run ./...
  trivy fs --scanners vuln,secret .
  ```
- Fix all findings before considering the task done.

### TypeScript / SvelteKit (`frontend/`)

- SvelteKit with `adapter-static`. Output must be a static bundle (no SSR).
- PWA via `@vite-pwa/sveltekit` with a custom `service-worker.ts` (injectManifest strategy).
- Offline sync: IndexedDB queue → Background Sync API → iOS `visibilitychange` fallback.
- Access tokens stored in memory (Svelte store) only. Refresh tokens in `localStorage`.
- After every change, run:
  ```
  npm run check
  npm run build
  npm audit
  ```
- Fix all high/critical findings reported by `npm audit` before considering the task done.
  Low-severity findings that have no available fix may be noted and left with an `overrides` pin.

### Shell scripts

- Run `shellcheck` on any shell script before considering the task done.

## Security Constraints

- `JWT_SECRET` env var must be ≥ 32 bytes; backend refuses to start otherwise.
- Passwords: minimum 16 characters enforced in the service layer on every create/reset.
- All SQL queries use `?` placeholders via `database/sql` — no string concatenation.
- Scorers may only POST hits to their computed score lane (`ownLane - 1`, wrapping).
  The lane is derived from the JWT identity server-side, never trusted from the client.
- Admins bypass lock checks and can overwrite any stage's hits directly.

## Domain Configuration

The Caddy domain is supplied via the `DOMAIN` environment variable (set in `.env` or
`docker-compose.yml`). The Caddyfile uses `{$DOMAIN}` substitution — no image rebuild
needed when changing domains.

## Rulebook Constants

Never hard-code shot counts or weapon class lists outside of `backend/internal/rulebook/`.
That package is the single source of truth for:
- Course-of-fire definitions (stages, shot counts, distances, times)
- Allowed weapon classes per discipline
- 5-shot revolver variant detection (`Standard Revolver 2.75" 5-Shot` → 40-shot 48-course)

## Docker

- `backend/Dockerfile`: multi-stage Go build, `CGO_ENABLED=0`, final image `alpine`.
- `frontend/Dockerfile`: Node build stage, copies output to `/srv/frontend`, then exits.
- `caddy` uses the stock `caddy:2-alpine` image with a mounted Caddyfile.
- Dockerfiles live in `backend/` and `frontend/`. `docker-compose.yml` lives in the repo root.
