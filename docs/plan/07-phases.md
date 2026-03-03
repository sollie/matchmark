# MatchMark — Implementation Phases

Each phase should be fully complete (linted, passing) before the next begins.
After every Go change: run `golangci-lint run ./...` and `trivy fs --scanners vuln,secret .`
After every shell script change: run `shellcheck`.

## Phase 1 — Backend Scaffold & Database

- [ ] Initialise Go module (`backend/go.mod`)
- [ ] Add dependencies: `chi`, `modernc.org/sqlite`, `golang-jwt/jwt/v5`,
      `golang.org/x/crypto`, `go-pdf/fpdf`
- [ ] Implement migration runner (`internal/db/migrate.go`) using `embed.FS`
- [ ] Write `001_init.sql` (full schema — see `02-database.md`)
- [ ] Implement admin seed (`internal/db/seed.go`): reads `ADMIN_PASSWORD` env,
      enforces ≥16 chars, skips if an admin already exists
- [ ] Validate required env vars at startup; refuse to start if missing or invalid
- [ ] Wire `cmd/server/main.go`: open DB → migrate → seed → start server

## Phase 2 — Authentication

- [ ] `internal/auth/password.go`: argon2id hash and verify
- [ ] `internal/auth/jwt.go`: sign/verify access token (1h HS256),
      sign/verify/rotate refresh token (7d, stored as SHA-256 hash)
- [ ] `internal/repository/user.go`: CRUD + refresh token table operations
- [ ] `internal/service/auth.go`: login, refresh, logout logic
- [ ] `internal/handler/auth.go`: POST /api/auth/login, /refresh, /logout
- [ ] `internal/auth/middleware.go`: RequireAuth + RequireAdmin chi middleware
- [ ] `internal/handler/user.go`: GET /api/users/me, admin CRUD + reset-password

## Phase 3 — Rulebook Package

- [ ] `internal/rulebook/courses.go`: stage definitions for 1500, 60, 48 disciplines
      (including 5-shot revolver variant detection)
- [ ] `internal/rulebook/weapons.go`: allowed weapon classes per discipline,
      `ValidateWeaponClass(discipline, weaponClass) error`

## Phase 4 — Competition & Shooter API

- [ ] `internal/repository/competition.go`, `shooter.go`, `stage.go`
- [ ] `internal/service/competition.go`: CRUD, status transitions
- [ ] `internal/service/shooter.go`: add/edit/delete shooter; eagerly create stage
      rows using rulebook data; validate weapon class + lane uniqueness
- [ ] `internal/handler/competition.go`: GET/POST/PUT /api/competitions[/{id}]
- [ ] `internal/handler/shooter.go`: GET/POST/PUT/DELETE
      /api/competitions/{id}/shooters[/{sid}]

## Phase 5 — Scoring API

- [ ] `internal/repository/hit.go`, `lock.go`
- [ ] `internal/service/scoring.go`:
  - `GetScoreLane(ownLane, lanesTotal int) int`
  - `GetStageDetail(...)`: resolve score lane, fetch stage + hits + lock state
  - `SubmitHits(...)`: validate ring values + count, replace hits in transaction,
    enforce lock check for scorers, bypass for admin
  - `LockStage(...)`: set locked=1, set fully_locked=1
  - `UnlockStage(...)`: admin only, reset both
- [ ] `internal/handler/scoring.go`:
  - GET  /api/competitions/{id}/lanes/{lane}/stage/{stage}
  - POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/hits
  - POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/lock
  - POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/unlock

## Phase 6 — Admin Monitoring (SSE)

- [ ] `internal/service/monitor.go`: `GetMonitorSnapshot(competitionID)` — single
      SQL query aggregating per-lane status, scores, lock state
- [ ] `internal/handler/monitor.go`: GET /api/admin/monitor?competition_id={id}
      — SSE stream; emit snapshot every 3s; handle client disconnect cleanly

## Phase 7 — Export

- [ ] `internal/export/json.go`: marshal per-shooter breakdown with per-ring counts
- [ ] `internal/export/csv.go`: flat row per shooter; headers in English
- [ ] `internal/export/pdf.go`: one page per shooter; scorecard layout with stage
      groupings per discipline; all labels in English
      (Stage, Points, X, 10, 9, 8, 7, 0, M, Scorer, Shooter, Match Director)
- [ ] `internal/handler/export.go`:
  - GET /api/competitions/{id}/export/pdf
  - GET /api/competitions/{id}/export/csv
  - GET /api/competitions/{id}/export/json

## Phase 8 — Frontend Shell

- [ ] Initialise SvelteKit project in `frontend/` with TypeScript + adapter-static
- [ ] Install `@vite-pwa/sveltekit`, configure `vite.config.ts` (see `04-frontend.md`)
- [ ] Write `service-worker.ts`: precache, NetworkFirst runtime cache for /api/competitions,
      Background Sync handler
- [ ] Create PWA icons (192×192, 512×512, maskable) and `manifest.webmanifest`
- [ ] Implement `lib/api/client.ts`: fetch wrapper with JWT attach + silent refresh
      + proactive refresh timer
- [ ] Implement `lib/stores/auth.ts`, `session.ts`, `scoring.ts`
- [ ] Implement `lib/db/idb.ts`: IndexedDB helper for sync queue
- [ ] Implement `lib/sync/queue.ts`: enqueue, replay, retry (max 3), iOS fallback
- [ ] `+layout.svelte`: auth guard (redirect to /login if no token); role-based redirect

## Phase 9 — Scorer UI

- [ ] `/login` page: username + password form; store tokens on success
- [ ] `/score/select` page: list active competitions; lane number input; derive and
      display score lane; navigate to `/score/stage` on confirm
- [ ] `/score/stage` page:
  - Stage progress dots
  - Shooter name + stage info header
  - Shot counter + running score + X count
  - Scoring keypad: X, 10, 9, 8, 7, 0 (grid) + M (full-width red)
  - Undo last hit
  - Previous / Next stage navigation
  - Lock Stage button (enabled only when shots == expected_shots)
  - Offline-safe: writes to IndexedDB first, syncs to backend
- [ ] `/score/summary` page: read-only per-stage breakdown; total score and X count

## Phase 10 — Admin UI

- [ ] `/admin` dashboard: links to users, competitions; active competition count
- [ ] `/admin/users` page: list users; create/edit/delete modal; reset password
- [ ] `/admin/competitions` page: list all competitions with status badge; create button
- [ ] `/admin/competitions/new` page: competition form + add shooters inline
- [ ] `/admin/competitions/[id]` page:
  - Competition detail + status controls (draft → active → finished)
  - Shooter list with edit/delete
  - Live SSE monitor table (lane, shooter, discipline, stage, score, X, locked)
  - Row click → slide-out detail panel with per-stage breakdown,
    Edit Hits modal (admin keypad, no lock enforcement), Unlock Stage button
- [ ] `/admin/competitions/[id]/export` page: PDF / CSV / JSON download buttons

## Phase 11 — Docker & Caddy

- [ ] `backend/Dockerfile`: multi-stage Go build; `CGO_ENABLED=0`
- [ ] `frontend/Dockerfile`: Node build stage; copy output to `/srv/frontend`
- [ ] `caddy/Caddyfile`: use `{$DOMAIN}` substitution; gzip; SPA fallback;
      /api/* reverse proxy to backend:8080
- [ ] `docker-compose.yml`: backend, frontend (copy-and-exit), caddy;
      named volumes: db_data, frontend_dist, caddy_data, caddy_config;
      `DOMAIN` env var passed to caddy
- [ ] `.env.example`: document all required env vars

## Phase 12 — Quality Assurance

- [ ] Run `golangci-lint run ./...` in `backend/`; fix all findings
- [ ] Run `trivy fs --scanners vuln,secret .` from repo root; address findings
- [ ] Manual PWA install test on Android (Chrome) and iOS (Safari)
- [ ] Offline scoring test: disable network mid-stage, complete stage, re-enable,
      verify sync queue replays correctly
- [ ] Admin SSE monitor test: verify reconnect after server restart
- [ ] Export test: generate PDF/CSV/JSON for a completed competition; verify layout

## Future Work (out of scope v1)

- 2FA / passkeys
- Rate limiting on auth endpoints
- Audit log (who changed what, when)
- Multi-language UI
- Re-entry and aggregate match types
