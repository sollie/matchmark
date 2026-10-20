# MatchMark v2 Spec

Scoring for WA1500/PPC competitions at a single club. Vocabulary is in `GLOSSARY.md`; decisions are in `docs/adr/`.

## Scope

In: competitions, a persistent shooter registry, lane-paired scoring devices, live spectator view, results report (`.xlsx`), internal detailed CSV, admin database backup.

Out for v1: classification and ranking, relays, Rifle, 60/48 beyond course data, offline operation, PDF/JSON export, multi-club hosting, accounts for scorers.

Future: relays; Rifle; DNS-01 and other ACME challenge methods; CDN-hosted frontend; multi-club tenancy; offline queue; classification.

## Users and flow

- Admin (password, cookie session): creates a competition, picks the course of fire, adds entries from the shooter registry (or creates new shooters), assigns lanes and categories, opens pairing, sets spectator visibility, watches progress, unlocks and corrects stages, finishes the competition, downloads the results report.
- Scorer (device): opens a pairing link once, then sees only "Scoring lane N: name", enters ring counts for the current stage and locks it.
- Spectator (no login): live standings and per-stage progress, subject to admin's per-competition field visibility.

## Scoring rules

- Score lane is the next manned lane after the scorer's own lane, wrapping, derived server-side from the device's pairing and current entries. A paired lane is always manned.
- A competition with one manned lane requires admin scoring.
- Everyone shoots in step. Entries change only between stages. A withdrawn entry is flagged and keeps its results; later stages re-target to the next manned lane.
- Hits per stage are seven counts (X, 10, 9, 8, 7, 0, miss) that must sum to the stage's expected shots. X scores 10. 0 and miss score 0.
- Lock stores the scorer's lane and name. Only the admin can unlock and edit; an admin edit of a locked stage keeps it locked.
- The admin entries list shows each stage's score and shots entered, for open as well as locked stages. An open stage with all expected shots entered is shown as "Scored, not locked".
- Course data and shot counts live in one `rulebook` package and match rulebook Issue 2026-01-01 §7. The 150 course is 10 stages grouped as Match 1 (7 yd, 15 yd), Match 2 (25 yd), Match 3 (50 yd), Match 4 (2 × 25 yd), Match 5 (7 yd, 25 yd, 50 yd, 25 yd × 6).

## Competition model

- Competition: name, date, course of fire (150 default; 60 and 48/40 selectable; no partial courses), status draft/active/finished, spectator visibility flags. A draft competition can be edited from its page (name, date and course); the course is locked once it has entries, and an active or finished competition cannot be edited.
- Shooter: last name, first name, sex, country, club, optional WAID. Persistent. A shooter without a WAID is a new shooter.
- Entry: shooter, category (Revolver, Pistol, Revolver OS, Pistol OS, Rifle), lane, withdrawn flag.
- Stage per entry, created when the entry is added: expected shots, seven ring counts, locked state, scorer.
- Device pairing: token, competition, lane, revoked flag.

## Results report

Filled in from the embedded `Results reporting mal.xlsx` using `excelize`. One sheet per category; competition name and date in the header; known shooters as WAID + total result, extra rows inserted as needed; new shooters in the "New shooters" block with last name, first name, sex, country, club, result. The template is a replaceable embedded file.

An internal detailed CSV lists ring counts per stage per entry for the club's own records. The CSV starts with a UTF-8 byte-order mark so spreadsheet applications show non-ASCII names correctly. Both downloads are named `<competition name>-<competition date>.<ext>` (for example `Test2-2026-10-20.xlsx`): characters other than letters, digits and `_` collapse to a single `-`, names are capped at 60 characters, an empty name becomes `results`, and non-ASCII letters are sent via an RFC 5987 `filename*`.

## API and frontend

JSON REST API under `/api`, documented with OpenAPI; admin and spectator endpoints separated; spectator endpoints are unauthenticated. Spectator, admin and scorer screens receive live updates over server-sent events (one stream per role and competition, see ADR 0010) instead of polling. Frontend is a Svelte + Vite + TypeScript SPA with a typed client generated from the OpenAPI spec, no service worker or IndexedDB. All strings in English.

`GET /api/rulebook` is read-only and unauthenticated. It returns the courses keyed `150`, `60`, `48` and `40` (label, discipline, total shots, stages with distance, expected shots, time and position), the weapon classes per discipline and the ring values. `internal/rulebook` is its only source, including the course labels; the SPA course dropdowns read from it.

## Backend

Go, chi, SQLite-format storage via Turso Database and `turso.tech/database/tursogo` (`CGO_ENABLED=0`, native library loaded at runtime; see ADR 0005), embedded ordered migrations, strict handler → service → repository layering, `excelize` for xlsx, `certmagic` for ACME. Admin password hashed with argon2id; session cookie `HttpOnly`, `SameSite=Lax`, `Secure` when `BASE_URL` is https.

## Configuration (environment only)

`ADMIN_PASSWORD` (≥16 chars), `DB_PATH`, `LISTEN_ADDR`, `BASE_URL`, `TRUSTED_PROXY`, `TLS_CERT_FILE`, `TLS_KEY_FILE`, `ACME_DOMAIN`, `ACME_EMAIL`, `TURSO_GO_CACHE_DIR` (writable path for the extracted native library), `REPLICATION_URL` (empty disables Replication), `REPLICATION_INTERVAL` (default 60s).

## Replication

Optional and off by default. When `REPLICATION_URL` is set, the primary opens its database through the tursogo sync API and pushes to a self-hosted `tursodb --sync-server` on another host, every `REPLICATION_INTERVAL` and when a competition finishes. Push-only: the replica is never merged back. Failures are logged and retried and never block scoring. The sync server has no authentication, so it must only be reachable over a private network; across the public internet use snapshot files instead. Restore is copying the replica's database file to a new primary. See ADR 0005.

TLS modes: plain HTTP (default, behind a proxy); static certificate; ACME via certmagic with certificates stored in the data volume.

## Score log

Whenever a device or the admin locks a stage, the server writes one JSON `info` line to stdout with `msg` and `event` `stage_locked`. Unlocking a stage writes a line with `event` `stage_unlocked` and the same fields. An admin overwrite of ring counts on a locked stage keeps it locked and takes it over: it writes a new `stage_locked` line with the new counts, `locked_by` `"Admin"` and `scorer_name` `"Admin"`, and the database records the same, so the original device lock is no longer visible; `locked_at` keeps the original lock time. The `stage_unlocked` line carries the `locked_by`, `scorer_name` and `locked_at` the stage had just before it was unlocked. Fields: `schema_version` (1), `event`, `emitted_at` (UTC, fixed-width millisecond precision such as `2026-10-19T16:45:14.123Z`), `seq` (integer that strictly increases for every line within a server process, assigned under a mutex so it matches write order), `competition` {`id`, `name`, `date`, `course_of_fire` {`key`, `discipline`, `total_shots`}}, `entry` {`id`, `lane`, `category`, `shooter` {`waid`, `first_name`, `last_name`, `club`}}, `stage` {`number`, `expected_shots`}, `ring_counts` (X, 10, 9, 8, 7, 0, miss), `locked_by` (lane number or `"Admin"`), `scorer_name`, `locked_at`. The line is written after the database write succeeds; a log failure never fails the lock. Editing a competition, an entry or a shooter re-emits `stage_locked` for every affected locked stage (a shooter edit covers all of that shooter's entries across competitions) with the `locked_by`, `scorer_name` and `locked_at` as currently recorded and a new `emitted_at` and `seq`; unlocked stages emit nothing. Replay: the last line per competition, entry and stage by (`emitted_at`, `seq`) wins; `seq` restarts at 1 after a server restart, which is safe because `emitted_at` is compared first, and a final `stage_unlocked` means the stage is not scored. Entries with no locked stage are not recoverable, and lines contain personal data. The `restore` subcommand (`matchmark restore [file]`, stdin when no file is given; no `ADMIN_PASSWORD` needed) parses the log, skips non-score and malformed lines (including an unknown course key, a `locked_by` that is neither a positive lane number nor `"Admin"`, or an `emitted_at` not in the fixed-width layout) with a count, rejects an unknown `schema_version`, applies the replay rule with context taken from the latest line, and prints a summary of the competitions, entries and scored stages. Without `--apply` it is a dry run and writes nothing. With `--apply` it opens the database from the server configuration (`DB_PATH`), runs migrations and recreates each competition (status `finished`), its shooters (matched by WAID, else by name and club), entries and scored stages with ring counts, lock state, `scorer_lane` (NULL for `Admin`), `scorer_name` and `locked_at`, keeping the original competition and entry ids. It refuses a database that already contains competitions or shooters and changes nothing; all competitions are written in one transaction, so a failure changes nothing; flags may follow the file path; `--merge` instead applies only data whose ids match or are new, lists conflicts (a differing competition, entry, or locked stage counts, `locked_by`, `scorer_name` or `locked_at`) and never overwrites them. It is idempotent by ids and leaves already locked stages untouched. Sex and country are not in the log, so restored shooters get sex `M` and an empty country. Unscored entries, devices and pairings are not restored and the command says so. See ADR 0007.

## Delivery

Repository root: `Dockerfile`, `docker-compose.yml` (app plus `./data` volume only), `docs/`, `GLOSSARY.md`, `README.md`, `AGENTS.md`. Code in `source/` (Go module root; `source/web/` for the SPA). Dockerfile: Alpine Node stage builds `web/`, Alpine Go stage embeds `web/dist` and builds with `CGO_ENABLED=0 -tags musl`, final Alpine 3.24 image (the tursogo native library needs a libc to `dlopen`) running non-root with `TURSO_GO_CACHE_DIR` on a writable path. Supported targets: linux/amd64 and linux/arm64 (64-bit Pi OS only); darwin/arm64 for development. The admin UI offers a database backup download; it uses `VACUUM INTO` with a string literal path, which Turso Database supports (bound parameters are rejected).

CI (GitHub Actions) builds, tests and smoke-tests the image natively: `ubuntu-latest` for amd64, `ubuntu-24.04-arm` for arm64, no QEMU, plus a build-only `macos-latest` job. The smoke test opens a database inside each image because a library-loading failure only shows at runtime.

## Rulebook verification

`docs/references/rulebook2026.txt` is a text extraction of the official PDF (Issue 2026-01-01, `https://wa1500.org/file.php?ID=1553`, SHA-256 `209f71e50b1e25abd167d4b2ea636e9f15be3b0c6d7303604dce3eaff003a123`). Re-extracted and compared on 2026-10-19: all 19,457 words match; only reading order of a few table labels and page numbers differs. The scorecard rows (X, 10, 9, 8, 7, 0, miss) and the course totals (24, 18, 24, 24 = 90; Match 5 = 60; 150) agree with v1's `rulebook.go`. Open: weapon-class naming for 60/48 courses (rulebook §3.17(g) says "2,5" 5-shot" where §3.10 says 2,75") is only relevant once 60/48 are used.
