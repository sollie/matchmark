# MatchMark — Implementation Phases

Each phase should be fully complete (linted, passing) before the next begins.
After every Go change: run `golangci-lint run ./...` and `trivy fs --scanners vuln,secret .`
After every shell script change: run `shellcheck`.

## Phase 1 — Backend Scaffold & Database

- [x] Initialise Go module (`backend/go.mod`)
- [x] Add dependencies: `chi`, `modernc.org/sqlite`, `golang-jwt/jwt/v5`,
      `golang.org/x/crypto`, `go-pdf/fpdf`
- [x] Implement migration runner (`internal/db/migrate.go`) using `embed.FS`
- [x] Write `001_init.sql` (full schema — see `02-database.md`)
- [x] Implement admin seed (`internal/db/seed.go`): reads `ADMIN_PASSWORD` env,
      enforces ≥16 chars, skips if an admin already exists
- [x] Validate required env vars at startup; refuse to start if missing or invalid
- [x] Wire `cmd/server/main.go`: open DB → migrate → seed → start server

## Phase 2 — Authentication

- [x] `internal/auth/password.go`: argon2id hash and verify
- [x] `internal/auth/jwt.go`: sign/verify access token (1h HS256),
      sign/verify/rotate refresh token (7d, stored as SHA-256 hash)
- [x] `internal/repository/user.go`: CRUD + refresh token table operations
- [x] `internal/service/auth.go`: login, refresh, logout logic
- [x] `internal/handler/auth.go`: POST /api/auth/login, /refresh, /logout
- [x] `internal/auth/middleware.go`: RequireAuth + RequireAdmin chi middleware
- [x] `internal/handler/user.go`: GET /api/users/me, admin CRUD + reset-password
- [x] `005_refresh_token_family.sql`: adds `family_id` column to `refresh_tokens`
      for detection of stolen/replayed refresh tokens

## Phase 3 — Rulebook Package

- [x] `internal/rulebook/courses.go`: stage definitions for 1500, 60, 48 disciplines
      (including 5-shot revolver variant detection)
- [x] `internal/rulebook/weapons.go`: allowed weapon classes per discipline,
      `ValidateWeaponClass(discipline, weaponClass) error`

## Phase 4 — Competition & Shooter API

- [x] `internal/repository/competition.go`, `shooter.go`, `stage.go`
- [x] `internal/service/competition.go`: CRUD, status transitions, spectator_enabled flag
- [x] `internal/service/shooter.go`: add/edit/delete shooter; eagerly create stage
      rows using rulebook data; validate weapon class + lane uniqueness
- [x] `internal/handler/competition.go`: GET/POST/PUT /api/competitions[/{id}]
- [x] `internal/handler/shooter.go`: GET/POST/PUT/DELETE
      /api/competitions/{id}/shooters[/{sid}]

## Phase 5 — Scoring API

- [x] `internal/repository/hit.go`, `lock.go`, `assignment.go`
- [x] `003_scorer_assignments.sql`: scorer assignment table
- [x] `internal/service/scoring.go`:
  - `GetStageDetail(...)`: fetch stage + hits + lock state (any authenticated user can read)
  - `SubmitHits(...)`: validate ring values + count, replace hits in transaction;
    non-admin must have a `scorer_assignment` whose `lane_number` matches the requested lane
    and stage must not be locked; admin bypasses both checks;
    returns `ErrSingleLaneAdminOnly` when competition has only one lane and caller is not admin
  - `LockStage(...)`: non-admin must be assigned to that lane; set locked=1, fully_locked=1;
    returns `ErrSingleLaneAdminOnly` for single-lane competitions (non-admin)
  - `UnlockStage(...)`: admin only, reset both
- [x] `internal/service/assignment.go`:
  - `SelectShooter(...)`: claim a shooter; enforce uniqueness per competition
  - `ReleaseAssignment(...)`: admin releases a shooter so another user can claim it
  - `GetMyAssignment(...)`: return caller's current assignment for session restore
  - `ListAssignments(...)`: return all assignments for a competition
- [x] `internal/handler/scoring.go`:
  - GET  /api/competitions/{id}/lanes/{lane}/stage/{stage}
  - POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/hits
  - POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/lock
  - POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/unlock  (admin)
  - Maps `ErrSingleLaneAdminOnly` → HTTP 403 `"admin scoring required"`
- [x] `internal/handler/assignment.go`:
  - POST   /api/competitions/{id}/assign
  - GET    /api/competitions/{id}/assign
  - GET    /api/competitions/{id}/assignments
  - DELETE /api/competitions/{id}/assign  (admin)

## Phase 6 — Admin Monitoring (SSE)

- [x] `internal/service/monitor.go`: `GetMonitorSnapshot(competitionID)` — single
      SQL query aggregating per-lane status, scores, lock state
- [x] `internal/handler/monitor.go`: GET /api/admin/competitions/{id}/monitor
      — SSE stream; authenticated via standard `Authorization: Bearer` header;
      emit snapshot every 3s; handle client disconnect cleanly
- [x] `008_drop_sse_tickets.sql`: drops the `sse_tickets` table (no longer needed)

## Phase 7 — Export

- [x] `internal/export/json.go`: marshal per-shooter breakdown with per-ring counts
- [x] `internal/export/csv.go`: flat row per shooter; headers in English
- [x] `internal/export/pdf.go`: one page per shooter; scorecard layout with stage
      groupings per discipline; all labels in English
      (Stage, Points, X, 10, 9, 8, 7, 0, M, Scorer, Shooter, Match Director)
- [x] `internal/handler/export.go`:
  - GET /api/competitions/{id}/export/pdf
  - GET /api/competitions/{id}/export/csv
  - GET /api/competitions/{id}/export/json

## Phase 8 — Frontend Shell

- [x] Initialise SvelteKit project in `frontend/` with TypeScript + adapter-static
- [x] Install `@vite-pwa/sveltekit`, configure `vite.config.ts` (see `04-frontend.md`)
- [x] Write `service-worker.ts`: precache, NetworkFirst runtime cache for /api/competitions,
      Background Sync handler
- [x] Create PWA icons (192×192, 512×512, maskable) and `manifest.webmanifest`
- [x] Implement `lib/api/client.ts`: fetch wrapper with JWT attach + silent refresh
      + proactive refresh timer
- [x] Implement `lib/stores/auth.ts`, `session.ts`, `scoring.ts`
- [x] Implement `lib/db/idb.ts`: IndexedDB helper for sync queue
- [x] Implement `lib/sync/queue.ts`: enqueue, replay, retry (max 3), iOS fallback
- [x] `+layout.svelte`: auth guard (redirect to /login if no token); role-based redirect

## Phase 9 — Scorer UI

- [x] `/login` page: username + password form; store tokens on success
- [x] `/score/select` page:
  - List active competitions
  - If caller already has an assignment (GET /assign), restore and navigate directly to stage
  - Otherwise: show the competition's shooter list; scorer taps their own name to claim it
    (POST /assign); on success navigate to `/score/stage`
- [x] `/score/stage` page:
  - Stage progress dots
  - Shooter name + stage info header
  - Shot counter + running score + X count
  - Scoring keypad: X, 10, 9, 8, 7, 0 (grid) + M (full-width red)
  - Undo last hit
  - Previous / Next stage navigation
  - Lock Stage button (enabled only when shots == expected_shots)
  - Offline-safe: writes to IndexedDB first, syncs to backend
  - Non-dismissible red banner + blocked keypad when API returns 403
    `"admin scoring required"` (single-lane competition)
- [x] `/score/summary` page: read-only per-stage breakdown; total score and X count

## Phase 9b — Post-Lock Stage Review

- [x] `internal/handler/scoring.go`: `StreamStage` — `GET /api/competitions/{id}/lanes/{lane}/stage/{stage}/stream`
      (authenticated via standard `Authorization: Bearer` header; verifies lane assignment
      on connect; streams until `fully_locked == true` then closes)
- [x] `cmd/server/main.go`: `StreamStage` registered inside `RequireAuth` group
- [x] `frontend/src/lib/api/sse.ts`: shared `openSSEStream()` utility using `fetch` +
      `ReadableStream` with SSE frame parser and `AbortController` for cancellation;
      replaces `EventSource` throughout the app (supports `Authorization: Bearer` header)
- [x] `/score/stage` page: post-lock review modal — spinner while adjacent scorer is pending,
      score/X/hits display once both sides locked, "Next →" blocked while pending,
      `goto('/score/summary')` moved from `lockStage()` into `nextStage()` so modal always shows
- [x] `golangci-lint run ./...` + `trivy fs --scanners vuln,secret .` — 0 findings
- [x] `npm run check && npm run build && npm audit` — 0 errors, 0 vulnerabilities

## Phase 10 — Admin UI

- [x] `/admin` dashboard: links to users, competitions; active competition count
- [x] `/admin/users` page: list users; create/edit/delete modal; reset password
- [x] `/admin/competitions` page: list all competitions with status badge; create button
- [x] `/admin/competitions/new` page: competition form + add shooters inline
- [x] `/admin/competitions/[id]` page:
  - Competition detail + status controls (draft → active → finished)
  - Shooter list with edit/delete
  - Live SSE monitor table (lane, shooter, discipline, stage, score, X, locked)
  - Row click → slide-out detail panel with per-stage breakdown,
    Edit Hits modal (admin keypad, no lock enforcement), Unlock Stage button
- [x] `/admin/competitions/[id]/export` page: PDF / CSV / JSON download buttons

## Phase 11 — Docker & Caddy

- [x] `backend/Dockerfile`: multi-stage Go build; `CGO_ENABLED=0`; accepts `ARG VERSION`
      and injects via `-X main.version=${VERSION}`; pinned `alpine:3.23.3` final image
- [x] `caddy/Dockerfile`: multi-stage build — Node stage builds SvelteKit, caddy stage
      copies output into the final image (replaces the old separate `frontend/Dockerfile`)
- [x] `caddy/Caddyfile`: uses `{$DOMAIN}` substitution; gzip; SPA fallback;
      /api/* reverse proxy to backend:8080
- [x] `caddy/Caddyfile.dev`: HTTP-only variant for local Docker dev (`localhost:8080`)
- [x] `caddy/Caddyfile.local`: local non-Docker dev via Caddy local CA (`https://localhost`);
      proxies /api/* to :8080, serves frontend from `../frontend/build`
- [x] `docker-compose.yml`: backend + caddy; named volumes: db_data, caddy_data, caddy_config;
      `DOMAIN` env var passed to caddy
- [x] `docker-compose.dev.yml`: local Docker dev override
- [x] `.env.example`: documents all required env vars

## Phase 12 — Quality Assurance

- [x] Run `golangci-lint run ./...` in `backend/`; fix all findings
- [x] Run `trivy fs --scanners vuln,secret .` from repo root; address findings
- [ ] Manual PWA install test on Android (Chrome) and iOS (Safari)
      Steps: deploy stack (`docker compose up`), open the URL in Chrome (Android) and
      Safari (iOS), accept the "Add to Home Screen" prompt, verify standalone mode
      launches with correct icon and no browser chrome.
- [ ] Offline scoring test: disable network mid-stage, complete stage, re-enable,
      verify sync queue replays correctly
      Steps: open `/score/stage` as a scorer, enable airplane mode, enter all hits for
      a stage and tap Submit — confirm the offline banner appears and hits are written to
      IndexedDB. Re-enable network and verify the sync queue drains and hits appear in
      the admin monitor.
- [ ] Admin SSE monitor test: verify reconnect after server restart
      Steps: open `/admin/competitions/[id]` as admin with an active competition, restart
      the backend container (`docker compose restart backend`), verify the monitor table
      reconnects automatically within ~30 s without a page reload.
- [ ] Export test: generate PDF/CSV/JSON for a completed competition; verify layout
      Steps: finish a competition with scored shooters, open `/admin/competitions/[id]/export`,
      download all three formats. Verify PDF has correct stage groupings (1500: Match 1–5;
      60/48: Subtotal 1–2 + Grand Total), all labels in English, and totals match CSV/JSON.

## Phase 14 — CI/CD Infrastructure

- [x] `.github/dependabot.yml`: weekly updates for GitHub Actions, Go modules, npm
- [x] `.github/CODEOWNERS`: `* @sollie`
- [x] `.github/workflows/ci.yml`: lint (`golangci-lint`) + type-check + build on every
      push/PR targeting main; Go and Node caches
- [x] `.github/workflows/codeql.yml`: CodeQL static analysis for Go and JavaScript
- [x] `.github/workflows/release.yml`: triggered on `v*` tags; cross-compiles backend
      for linux/amd64 and linux/arm64; attaches binaries to a GitHub release via
      `softprops/action-gh-release` (pinned to full commit SHA)

## Future Work (out of scope v1)

- 2FA / passkeys
- Rate limiting on auth endpoints
- Audit log (who changed what, when)
- Multi-language UI
- Re-entry and aggregate match types

## Phase 13 — Spectator Live View

Backend:
- [x] Write `002_spectator.sql`: `ALTER TABLE competitions ADD COLUMN spectator_enabled INTEGER NOT NULL DEFAULT 0`
- [x] Update `internal/model/model.go`: add `SpectatorEnabled bool` to Competition struct
- [x] Update `internal/repository/competition.go`: include `spectator_enabled` in all
      queries; support filtering by `spectator_enabled=1` in spectator list query
- [x] Update `internal/service/competition.go`: accept `spectator_enabled` in create/update
- [x] `internal/service/spectator.go`: `GetActiveSpectatorCompetitions()`,
      `GetSpectatorSnapshot(competitionID)` — returns 404-sentinel if not found or flag off
- [x] `internal/handler/spectator.go`:
      `GET /api/spectator/competitions` (no auth)
      `GET /api/spectator/competitions/{id}/live` (SSE, no auth; 3s interval; clean disconnect)
- [x] Wire new routes in `main.go` outside the `RequireAuth` group
- [x] `golangci-lint run ./...` + `trivy fs --scanners vuln,secret .`; fix all findings

Frontend:
- [x] `/live` route: fetch spectator competition list; display as selectable cards; no auth guard
- [x] `/live/[id]` route: SSE scoreboard; live ranked table; dynamic per-stage columns;
      auto-reconnect with exponential backoff; 404 error state
- [x] Exclude `/live` and `/live/[id]` from auth guard in `+layout.svelte`
- [x] `npm run check && npm run build`; fix all findings
