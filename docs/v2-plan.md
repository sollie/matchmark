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
- [ ] 10b. Optional push-only Replication (timer plus on competition finish), `REPLICATION_*` config, `tursodb --sync-server` compose snippet and private-network warning in `README.md`.
- [x] 11. Root `Dockerfile` (Alpine Node and Go stages, Alpine 3.24 final with `-tags musl`, non-root, `TURSO_GO_CACHE_DIR`), `docker-compose.yml`, `README.md`, and native-runner CI (amd64, arm64, darwin/arm64 build-only) with an in-image database smoke test.
- [ ] 12. Manual QA: full flow on a phone, results report opened in Excel, Pi run. Then delete `docs/plan/`, update `AGENTS.md`, `docs/STATUS.md`, and merge.
