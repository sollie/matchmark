# Copilot Instructions — MatchMark

Single-club WA1500/PPC competition scoring system: one Go binary (`source/`) with an embedded Svelte SPA (`source/web/`). The spec is `docs/spec.md`, vocabulary is `GLOSSARY.md`, decisions are in `docs/adr/`; `AGENTS.md` holds the binding agent rules.

## Architecture

- **Backend layering is strict:** `handler` → `service` → `repository` (under `source/internal/`). No business logic in handlers or repositories. Wiring, config, TLS and request logging are in `cmd/server/`.
- **Storage:** SQLite format via `turso.tech/database/tursogo` (ADR 0005); schema evolves through ordered migrations in `internal/db/migrations/` (never edit an applied migration). Optional push-only replication.
- **`internal/rulebook/` is the single source of truth** for courses of fire and shot counts. Never hard-code them elsewhere.
- **Scoring model:** the score lane is the next manned lane after the scorer's own lane, wrapping, derived server-side from the device pairing and current entries. A competition with one manned lane requires admin scoring. Hits are seven ring counts per stage that must sum to the expected shots. A lock records the scorer; only the admin unlocks, edits, or locks as "Admin".
- **Auth:** admin password from `ADMIN_PASSWORD` and a cookie session with login rate limiting. Scorers are paired devices (QR link), not accounts. Spectator endpoints are unauthenticated and gated by per-competition visibility flags.
- **Reports:** `.xlsx` filled from the embedded template via excelize, plus a detailed CSV.
- **Logging:** JSON `log/slog` to stdout, one `request` line per HTTP request.
- **Docker:** one image built from the root `Dockerfile`; TLS is handled by the binary (plain, static, or certmagic).

## Rules

Build/test commands, conventions, git workflow and documentation rules are in `AGENTS.md`; follow it.
