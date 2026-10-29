# v2 Rewrite Plan

Work on branch `v2`. One step per commit, with checks run before each. Tick a step when its commit lands.

- [x] 1. Move `backend/` to `source/` (Go module root), keep `internal/rulebook`; remove `caddy/`, `frontend/` (old), PDF/JSON export, SSE, JWT and refresh-token code once replaced. Delete tracked CodeQL artifacts (`codeql-db-*`, `results-*`).
- [x] 2. Schema: new migrations for shooters, competitions, entries, stages (ring counts), device pairings, admin session. Service tests for score-lane derivation (empty lanes, wraparound, single lane, withdrawn entries) and ring-count validation.
- [x] 3. Admin auth: password from env, cookie session, rate limiting on login.
- [x] 4. Competition, shooter registry, entry and lane management APIs.
- [x] 5. Device pairing and scorer API (score lane derived server-side, lock, admin unlock).
- [x] 6. Spectator API with per-competition visibility flags.
- [x] 7. Results report: embed template, generate `.xlsx` with excelize; detailed CSV. Tests against the template.
- [x] 8. OpenAPI spec and generated TypeScript client.
- [x] 9. `source/web/`: Svelte + Vite SPA: admin, scorer, spectator screens.
- [x] 10. Embed `web/dist`, TLS modes (plain, static, certmagic), backup endpoint.
- [x] 10a. Spike: swap `modernc.org/sqlite` for tursogo (ADR 0005). Run `001_schema.sql` and the full test suite, check `foreign_keys`, `busy_timeout`, `SetMaxOpenConns(1)` and the backup download. Stop and revisit the ADR if it fails. Result: passes on tursogo v0.8.3; `foreign_keys` is not on by default and is set per connection, `VACUUM INTO` works only with a string literal path (no bound parameter).
- [x] 10b. Optional push-only Replication (timer plus on competition finish), `REPLICATION_*` config, `tursodb --sync-server` compose snippet and private-network warning in `README.md`.
- [x] 11. Root `Dockerfile` (Alpine Node and Go stages, Alpine 3.24 final with `-tags musl`, non-root, `TURSO_GO_CACHE_DIR`), `docker-compose.yml`, `README.md`, and native-runner CI (amd64, arm64, darwin/arm64 build-only) with an in-image database smoke test.
- [x] 12. Manual QA: full flow on a phone, results report opened in Excel, Pi run. Then delete `docs/plan/`, update `AGENTS.md`, `docs/STATUS.md`, and merge.
- [x] 13. `GET /api/rulebook` (#140): courses with labels, weapon classes and ring values from `internal/rulebook`; the SPA course dropdowns read from it.

## Live updates (ADR 0010)

- [x] 14. Broadcaster in `service` (per-competition topic, 250 ms coalescing, drop slow subscribers) fired after each successful write; shared SSE writer in `handler` (heartbeat 25 s, 30 min lifetime, global cap 500, `Cache-Control: no-cache`, `X-Accel-Buffering: no`, flushing through compression). Tests for broadcaster, stream handler and compression flush.
- [x] 15. Shared `fetch`-based SSE client in the SPA (backoff reconnect, close while the tab is hidden, stale indicator, terminal `revoked`). Vitest for parser and reconnect.
- [x] 16. Spectator stream (`snapshot` with standings and progress, built once per change) replacing the 5 s and 30 s polling; OpenAPI `text/event-stream`; e2e asserts a change appears without a poll.
- [x] 17. Admin stream (`changed` ping, SPA refetches entries) replacing the 3 s polling.
- [x] 18. Scorer stream (`snapshot` with the session, `revoked` on unpair or finish; the unsaved ring counts are kept, a changed score lane shows a notice and resets them, a stage that is not open disables save; later extended with the server-derived `phase`, see ADR 0013) via the device-token header.
- [x] 19. Update `README.md` (reverse proxies must not buffer `text/event-stream`), `docs/spec.md` (SSE no longer out, polling text) and set ADR 0010 to accepted.

## Release pipeline (ADR 0012)

- [x] 20. Version stamping, `.goreleaser.yaml`, reusable test and image workflows, tag-triggered release with GHCR multi-arch images, Dependabot for npm and Docker, README and ADR. Validate with a `v0.1.0-rc.1` tag after merging.

## Shooter acceptance (ADR 0013)

- [x] 21. Stage status model, `stage_status` Score log v2 and restore (#235).
- [x] 22. Shooter acceptance and dispute from the device, with the server-derived device phase (#236).
- [x] 23. Lockstep on a current stage, the gate and auto-finish (#237).
- [x] 24. Admin control room (#238) and dispute history (#239).
- [x] 25. Mid-competition restore (#240) and the documentation sweep (#241).
- [x] 26. Rename the Rifle category to Carbine (#338) and allow it on the 150-shot course (#339).

## Competition contains Matches (ADR 0017)

Delivered as one stack of PRs that merge together (#345). Until step 28 lands, the code and the rest of the docs still use "competition" for what is now a Match.

- [x] 27. Docs: glossary, spec and ADR 0017 (#346).
- [x] 28. Squash migrations into the `matches` and `competitions` schema (#347).
- [x] 29. Rename Competition to Match across backend, API, Score log v3 and SPA (#348).
- [x] 30. Competition container CRUD with date-range validation (#349).
- [ ] 31. Part labels from `internal/rulebook` in the UI and reports (#350).
- [x] 32. Competition-level zip downloads (#351).
- [x] 33. Scoring segments: ADR 0021, glossary, rulebook segments served by `GET /api/rulebook` (#423).
- [x] 34. Segment backend state and services: storage, status, lockstep and Score log (#424).
- [ ] 35. Scorer, admin and spectator UI per segment (#425).
- [ ] 36. Scorecard copy, results and reports with one column per segment (#426).
