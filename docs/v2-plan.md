# v2 Rewrite Plan

Work on branch `v2`. One step per commit, with checks run before each. Tick a step when its commit lands.

- [ ] 1. Move `backend/` to `source/` (Go module root), keep `internal/rulebook`; remove `caddy/`, `frontend/` (old), PDF/JSON export, SSE, JWT and refresh-token code once replaced. Delete tracked CodeQL artifacts (`codeql-db-*`, `results-*`).
- [ ] 2. Schema: new migrations for shooters, competitions, entries, stages (ring counts), device pairings, admin session. Service tests for score-lane derivation (empty lanes, wraparound, single lane, withdrawn entries) and ring-count validation.
- [ ] 3. Admin auth: password from env, cookie session, rate limiting on login.
- [ ] 4. Competition, shooter registry, entry and lane management APIs.
- [ ] 5. Device pairing and scorer API (score lane derived server-side, lock, admin unlock).
- [ ] 6. Spectator API with per-competition visibility flags.
- [ ] 7. Results report: embed template, generate `.xlsx` with excelize; detailed CSV. Tests against the template.
- [ ] 8. OpenAPI spec and generated TypeScript client.
- [ ] 9. `source/web/`: Svelte + Vite SPA: admin, scorer, spectator screens.
- [ ] 10. Embed `web/dist`, TLS modes (plain, static, certmagic), backup endpoint.
- [ ] 11. Root `Dockerfile` (Alpine Node and Go stages, scratch final, CA certs, non-root), `docker-compose.yml`, `README.md`.
- [ ] 12. Manual QA: full flow on a phone, results report opened in Excel, Pi run. Then delete `docs/plan/`, update `AGENTS.md`, `docs/STATUS.md`, and merge.
