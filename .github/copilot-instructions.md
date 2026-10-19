# Copilot Instructions — MatchMark

WA1500/PPC competition scoring system: Go REST API (`backend/`) + SvelteKit PWA (`frontend/`) + Caddy (`caddy/`). The spec lives in `docs/plan/` (start at `00-index.md`); `AGENTS.md` holds the binding agent rules. Current status/open work: `docs/STATUS.md`.

## Build, test, lint

Backend (run in `backend/`, always `CGO_ENABLED=0`):
```sh
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test ./internal/service -run TestGetScoreLane -v   # single test
golangci-lint run ./...
trivy fs --scanners vuln,secret .                                   # from repo root
go run ./cmd/server    # needs JWT_SECRET (>=32 bytes), ADMIN_PASSWORD (>=16 chars), DB_PATH
```

Frontend (run in `frontend/`):
```sh
npm run check && npm run build && npm audit   # required after every change; no test runner exists
```

Shell scripts: `shellcheck`. Full stack: `docker compose up --build` (dev, plain HTTP: add `-f docker-compose.dev.yml`). Local CodeQL: `./scripts/codeql-local.sh`.

## Architecture

- **Backend layering is strict:** `handler` → `service` → `repository` (under `backend/internal/`). No business logic in handlers or repositories. Wiring/DI, middleware, rate limiters and routes are all in `cmd/server/main.go`.
- **SQLite** via pure-Go `modernc.org/sqlite`; schema evolves through embedded, ordered migrations in `internal/db/migrations/` (never edit an applied migration; add a new numbered one). Migrations 004/006/007 created `sse_tickets`, 008 dropped it.
- **`internal/rulebook/` is the single source of truth** for courses of fire, shot counts and weapon classes. Never hard-code these elsewhere in Go. The frontend mirror is `frontend/src/lib/rulebook.ts`. `Standard Revolver 2.75" 5-Shot` auto-selects the 40-shot 48-course variant.
- **Stages are created eagerly** when a shooter is added; submitting hits replaces all hits for that stage in one transaction (no patch/undo endpoints).
- **Scoring model:** each lane has one scorer who scores the neighbouring lane: `scoreLane = ownLane - 1` (lane 1 wraps to `lanes_total`). It is derived server-side from the JWT/assignment, never from client input. Single-lane competitions require admin scoring (`ErrSingleLaneAdminOnly` → 403 `"admin scoring required"`). Admins bypass lock and assignment checks.
- **Auth:** HS256 access token (1h, in-memory only on the client, `iss=matchmark`, `aud=matchmark-api`); refresh token (7d, rotated, family-tracked, stored hashed in DB) lives only in the `mm_rt` HttpOnly cookie. `users.auth_version` revokes sessions. Client IP comes from a custom `trustedProxyRealIP` that trusts forwarded headers only from private/loopback peers (`TRUSTED_PROXY`).
- **Realtime:** admin monitor and spectator views are SSE (3s snapshot polling). The frontend uses `lib/api/sse.ts` (`fetch` + `ReadableStream`, not `EventSource`) so the `Authorization` header works. Spectator endpoints (`/api/spectator/...`) are unauthenticated and gated by the competition's `spectator_enabled` flag.
- **Offline sync (frontend):** scorer writes go to IndexedDB first (`lib/db/idb.ts`), replayed by `lib/sync/queue.ts` and the custom `service-worker.ts` (Background Sync; `visibilitychange` fallback for iOS). Static adapter only — no SSR.
- **Docker:** two services only (`backend`, `caddy`). The caddy image builds the SvelteKit app in a Node stage. Domain comes from `DOMAIN` via `{$DOMAIN}` in the Caddyfile.

## Conventions

- **Never add comments to code.**
- All strings (UI, PDF, API fields, logs) in English only.
- All SQL uses `?` placeholders; no string concatenation.
- Input limits are validated in handlers (username ≤ 64, name ≤ 128, wa_id ≤ 32, club ≤ 128); passwords ≥ 16 chars enforced in the service layer on every create/reset.
- Tests are table/unit tests in `internal/service`, `internal/auth`, `internal/rulebook` and `cmd/server`; follow the existing style when adding service tests.
- Use the latest stable dependency versions; verify upstream before pinning; remove unused deps. Fix high/critical `npm audit` findings.
- Keep `docs/plan/` and `README.md` in sync with any change to architecture, API, Docker or run/deploy steps; tick items in `docs/plan/07-phases.md` when done.
- Do not run git write operations (commit/push/tag). After finishing a change, suggest a commit message. When working through a list of items (reviews), finish one, run checks, suggest the commit message, then stop and wait.
