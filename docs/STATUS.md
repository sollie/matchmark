# MatchMark — Project Status & Restart Checklist

Snapshot taken 2026-10-19. Last commit: 2026-03-06 (59 commits, working tree clean).
Purpose: sanity-check the project when development resumes after a long idle period.

## 1. Where we are

Feature-complete for v1 per `docs/plan/07-phases.md`. Phases 1–11, 13, 14, 15 are fully checked off.
Only **Phase 12 manual QA** is open. Recent work (Mar 2026) was security hardening: auth session
revocation (`auth_version`, migration 009), scorer/shooter access scoping, per-IP rate limiting,
HttpOnly refresh cookie, SSE via fetch streams, JWT iss/aud, CodeQL local script.

### Verified on restart (2026-10-19)

| Check | Result |
|-------|--------|
| `go test ./...` (backend) | Pass (cmd/server, auth, rulebook, service) |
| `golangci-lint run ./...` | 0 issues |
| `npm run check` | 0 errors, 0 warnings |
| `npm run build` | Success |
| `npm audit` | **19 vulns (12 high, 5 moderate, 2 low)** — violates AGENTS.md rule |
| `trivy fs --scanners vuln,secret .` | Not run (trivy not found on PATH) |
| Go toolchain locally | 1.27.2 (AGENTS.md says 1.26+; check `go.mod` / Dockerfile / CI agree) |

## 2. What remains

### Must do before using in production
1. **Fix `npm audit`** — run `npm audit fix` in `frontend/` (e.g. vite 7.3.1 → 7.3.7 fixes the flagged advisory);
   re-run `check` + `build`. Pending minor bumps: kit 2.53→2.70, svelte 5.53→5.57, svelte-check, idb 8.0.4.
2. **Refresh Go deps** — many have newer versions (e.g. `golang.org/x/crypto` 0.48→0.58, `x/net`, chi 5.2.5→5.5.0).
   Install/run `trivy` and re-scan. Check Dependabot PRs on GitHub (likely a backlog).
3. **Phase 12 manual QA (all still unchecked):**
   - PWA install on Android Chrome + iOS Safari (standalone, icon)
   - Offline scoring: airplane mode mid-stage, verify queue drains and admin monitor updates
   - Admin SSE monitor reconnect after `docker compose restart backend` (~30 s, no reload)
   - Export test: PDF (1500: Match 1–5; 60/48: Subtotal 1–2 + Grand Total), CSV, JSON totals agree
4. **Real-world smoke test** — full flow with `docker compose up --build`: seed admin → create users →
   create competition + shooters → scorers claim → score → lock → review modal → export → spectator `/live`.
5. **Check CI state** on GitHub (`ci.yml`, `codeql.yml`) — confirm still green after dependency/runner drift.

### Test coverage gaps
- No tests in `handler`, `repository`, `db`, `export` packages; no frontend tests.
- Export (PDF/CSV/JSON) and offline sync queue/service worker are only covered by manual QA.
- Consider a handler-level integration test with in-memory SQLite for the scoring happy path.

### Documentation drift (fix `docs/plan/` per AGENTS.md)
- `07-phases.md` Future Work lists **"Rate limiting on auth endpoints"** as out of scope, but it is implemented
  (`cmd/server/main.go`: auth 5 rps/burst 10, API 60 rps/burst 120). Move to done / add to a phase; document in `03-backend.md`.
- `07-phases.md` Phase 3 names `rulebook/courses.go` and `weapons.go`; actual code is a single `rulebook.go`.
- `01-architecture.md` / `02-database.md`: confirm they mention migrations 004/006/007 (sse_tickets, later dropped by 008)
  and `009_user_auth_version.sql` / `users.auth_version` semantics.
- `03-backend.md`: verify auth contract documents session revocation via `auth_version`, scorer/shooter scoping,
  and `TRUSTED_PROXY` env var.
- `docker-compose.yml` does not pass `FRONTEND_ORIGIN` or `TRUSTED_PROXY` consistently with `.env.example` /
  README (README documents `FRONTEND_ORIGIN`; compose does not forward it). Decide and align.
- Phase ordering in `07-phases.md` is out of sequence (13 after 15; 9b); tidy.
- README says compose "orchestrates all three services" — it is two (backend, caddy).
- Phase 12 trivy/lint checkmarks are stale; re-tick after this restart run.

### Repo hygiene
- Untracked-but-ignored CodeQL artifacts sit in the repo root (`codeql-db-go`, `codeql-db-js`, `results-*.csv/sarif`);
  confirm they're gitignored everywhere (`git check-ignore` confirms db-go and results-go.csv) and delete if unneeded.
- `docs/references/` holds `rulebook2026.txt` and `spec-bundle.yaml`; check the WA1500 rulebook for a newer edition
  and re-verify `backend/internal/rulebook/` and `05-rulebook.md` against it.

## 3. Suggested restart order
1. `git pull`; check GitHub (Dependabot PRs, CI, CodeQL alerts).
2. Update deps (npm + Go), run: `golangci-lint run ./...`, `go test ./...`, `trivy fs --scanners vuln,secret .`,
   `npm run check && npm run build && npm audit`.
3. Fix doc drift above.
4. Deploy locally with Docker and run the Phase 12 manual QA; tick boxes.
5. Decide on Future Work (2FA/passkeys, audit log, multi-language UI, re-entry/aggregate match types).

## 4. Key invariants to keep in mind (from AGENTS.md / plan)
- Scorer score-lane = `ownLane - 1` (wraps), derived server-side from JWT; single-lane comps require admin scoring.
- Rulebook constants only in `backend/internal/rulebook/`; 5-shot revolver → 40-shot 48-course.
- Refresh token only in `mm_rt` HttpOnly cookie; access token in memory only.
- `JWT_SECRET` ≥ 32 bytes, passwords ≥ 16 chars; no comments in code; English-only strings; no git writes by the agent.
