# MatchMark

WA1500/PPC competition scoring system: one Go binary (REST API + embedded Svelte SPA) backed by a single SQLite file. See [`docs/spec.md`](docs/spec.md) and [`docs/v2-plan.md`](docs/v2-plan.md).

The rulebook (courses, weapon classes, ring values) is public at `GET /api/rulebook`; the SPA reads its course labels from it. The build version is public at `GET /api/version` and shown in the SPA footer next to a link to the repository.

## Layout

| Path | Description |
|------|-------------|
| `source/` | Go module (`cmd/server`, `internal/`, `api/openapi.yaml`) |
| `source/web/` | Svelte + Vite + TypeScript SPA, built to `source/web/dist` and embedded in the binary |
| `Dockerfile` | Node build stage, Go build stage, Alpine 3.24 final image built with `-tags musl` (non-root) |
| `docker-compose.yml` | The app plus a `./data` volume |
| `scripts/smoke-image.sh` | Starts an image and checks it opens its database |
| `.goreleaser.yaml`, `.github/workflows/` | Release binaries, CI, tag-triggered release (ADR 0012) |
| `docs/` | Spec, ADRs, plan |

## Run with Docker

```sh
mkdir -p data && sudo chown 65532:65532 data
export ADMIN_USERNAME='admin'
export ADMIN_PASSWORD='at-least-16-characters'
docker compose up --build -d
```

The two `ADMIN_` variables are only needed for the first start; later starts work without them.

The app listens on plain HTTP at `http://localhost:8080` (override the host port with `HTTP_PORT`). The container runs as UID/GID 65532, so the `data` directory must be writable by that user. The database (a SQLite-format file opened through the tursogo driver), backups and ACME certificates live in `/data`.

Sign in at `/` with the seeded admin's username and password. On first start, when there are no users, an admin is created from `ADMIN_USERNAME` and `ADMIN_PASSWORD`; the server refuses to start if either is missing or invalid. Once any user exists both variables are ignored.

## Run without Docker

Prerequisites: Go 1.27+ and Node.js 24 LTS.

```sh
cd source/web && npm ci && npm run build && cd ..
ADMIN_USERNAME='admin' ADMIN_PASSWORD='at-least-16-characters' CGO_ENABLED=0 go run ./cmd/server
```

For frontend development run `npm run dev` in `source/web`; it proxies `/api` to `localhost:8080`. Run the frontend tests with `npm test` (Vitest, jsdom and Testing Library; dev-only).

## End-to-end tests

Playwright drives the real binary with the embedded SPA in Chromium, Firefox and WebKit (ADR 0008). Each run starts a fresh server with a temporary database.

```sh
cd source/web
npm ci && npm run build
npx playwright install chromium firefox webkit
npm run e2e
npm run e2e -- --project=firefox
```

Without local browsers or Go, run the suite in Docker against the built image:

```sh
docker compose -f docker-compose.e2e.yml up --build --abort-on-container-exit --exit-code-from e2e
```

Set `E2E_BASE_URL` (and `E2E_ADMIN_USERNAME` and `E2E_ADMIN_PASSWORD`) to test an already running instance instead of starting one. CI runs one job per browser in parallel on every pull request, gated by an aggregate `E2E` check, and uploads a per-browser report on failure.

Coverage, by area:

- Password change: wrong current password, other sessions ended, change and revert.
- Users: an admin creates a user with a temporary password, the user is forced to change it at first login and then works as a regular user without the Users or Backup links. Admins can allow or forbid backup downloads per admin (the Backup link shows only for admins with the permission), and can also deactivate and reactivate users, reset a password to a new temporary one, and promote or demote; deactivation, reset and demotion end that user's sessions, and the last active admin is protected.
- Scorecard copy: once a competition is finished, anyone can open `/api/scorer/scorecard/<competition id>/<lane>` (add `?download=1` to download) to get a PNG of that lane's scorecard. It is public, ignores the spectator visibility flags, is rendered on demand and never stored (ADRs 0015 and 0016). The server limits renders (4 at a time, 20 requests per minute per client IP and card, then `429` or `503` with `Retry-After`); set `TRUSTED_PROXY` so the real client IP is used. Ids are sequential, so add per-IP rate limiting at your CDN or reverse proxy edge (for example a Cloudflare rate limiting rule on `/api/scorer/scorecard/*`) and do not cache by file extension; responses send `Cache-Control: private, no-cache`.
- Club settings: an admin sets the club name and uploads or removes a PNG or JPEG logo (at most 512 KB and 2048 by 2048 pixels) at `/admin/club`; the name shows on the public page, and regular users never see the screen.
- Audit log: an admin finds entries filtered by user, action and date and pages through them; regular users never see the screen.
- Auth: wrong password, unknown username and correct login, per-client login lockout, sign out, 401 on unauthenticated admin API calls, the not-found page, and the unpaired scorer page.
- Shooter search: picking a shooter for an entry through the search combobox, the no-match state and hiding shooters already entered.
- Shooters: add, edit, cancel an edit, delete with confirmation, form validation, local search, and WA1500 import (sex selector, name-conflict dialog, manual fallback; the WA1500 search is mocked).
- Competitions: creation for each course of fire with its stage columns, the prefilled ISO 8601 date and invalid-date blocking, the list with statuses, start and finish with confirmation, finished competitions (revoked pairings, no new entries), editing a draft (name, date, course, locked course once entries exist, no edit once started), delete with confirmation, and visibility flags persisting.
- Entries and admin scoring: adding entries with category and default lane, taken-lane rejection, move, withdraw and remove, live shot totals with rejection of wrong totals, and save and lock as the signed-in user (an accepted stage), accepting a stage that awaits acceptance, and unlock.
- Dispute history: the stage event table with scorer, shooter and stage filters and the dispute count per scorer.
- Control room: the lane grid with scorer, shooter, status, disputes, paired and best-effort connected, Start next stage and Accept all enabled only when allowed, and per-row Accept and Unlock for an unpaired lane.
- Lockstep: the next-stage gate, devices limited to the current stage, the next-stage banner, auto-finish with the finished screen, and the frozen entry list.
- Pairing and scorer device: pairing link creation, the device's lane and score lane, score lane wrap-around, running total and mismatch message, save and lock (the stage awaits acceptance), the waiting screen, the shooter reviewing, accepting or disputing their result with the Disputed notice and prefilled values, read-only locked stages, revoked pairings, the admin-scoring requirement with one manned lane, unpairing a device, and refusal to score a draft competition.
- Spectator: ranking by total then X count with the stage status row, draft and finished visibility, each visibility flag, the empty state, and 404 for a draft standings API.
- Results view: finished competitions only, deep link, visibility flags, ranking ties and the 404 for unfinished ones.
- Reports and backup: `.xlsx` results report and detailed CSV (downloaded from the competition list, finished competitions only), database backup, and 401 without a session.

Not covered by the browser suite: the score log (Go tests, ADR 0007), accessible-name cleanup (#129), and phone and tablet viewports.

Tests seed data through the admin API with unique names and send a unique `X-Forwarded-For` per client so each one has its own login rate limit. The e2e server trusts `127.0.0.1`, and `docker-compose.e2e.yml` sets `TRUSTED_PROXY` to the private network ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), so the e2e container's peer address is trusted and its forwarded headers are honoured.


## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_USERNAME` | required when there are no users | Username of the seeded admin: 3-32 characters from `a-z`, `0-9`, `.`, `_`, `-` (stored lowercase); ignored once users exist |
| `ADMIN_PASSWORD` | required when there are no users | Password of the seeded admin, 16-128 characters; ignored once users exist |
| `DB_PATH` | `data/matchmark.db` (`/data/matchmark.db` in the image) | SQLite file; its directory also holds backups and ACME certificates |
| `LISTEN_ADDR` | `:8080` plain, `:8443` with TLS | Listen address |
| `BASE_URL` | empty | Public URL, used for pairing links; an `https` URL marks the session cookie `Secure` (see [TLS modes](#tls-modes)) |
| `TRUSTED_PROXY` | empty | Comma-separated IPs or CIDRs of reverse proxies whose forwarded client IP headers are trusted |
| `DISABLE_COMPRESSION` | `false` | Set to `true` to turn off response compression (for example behind a proxy that already compresses) |
| `TLS_CERT_FILE`, `TLS_KEY_FILE` | empty | Static certificate mode; set both |
| `ACME_DOMAIN`, `ACME_EMAIL` | empty | Automatic certificates (Let's Encrypt); both required together |
| `REPLICATION_URL` | empty | Base URL of a `tursodb --sync-server`; empty disables [Replication](#replication) |
| `REPLICATION_INTERVAL` | `60s` | Push interval as a Go duration (for example `30s`, `5m`); must be positive |

Static and ACME modes are mutually exclusive.

## WA1500 lookup

Admins can look up shooters in the public WA1500 register (`https://wa1500.org`, fixed host, no configuration) with `GET /api/wa1500/search?q=` and import a hit with `POST /api/shooters/import`. The server needs outbound HTTPS to `wa1500.org`; without it the lookup answers 502 and shooters are entered by hand. See ADR 0014 and `docs/spec.md`.

## Logging

Text responses (SPA assets, JSON, CSV) of at least 256 bytes are compressed with Brotli, gzip or deflate according to the request `Accept-Encoding` (preference: `br`, `gzip`, `deflate`). `.xlsx` reports, other binary types, HEAD and Range requests are left alone. Browsers only offer Brotli over HTTPS.

The server writes structured JSON logs (`log/slog`) to stdout: one `request` line per HTTP request with `method`, `path` (no query string), `status`, `bytes`, `duration_ms`, `remote` (the client IP, honouring `TRUSTED_PROXY`) and `user_agent`. Level is `INFO` for 1xx-3xx, `WARN` for 4xx and `ERROR` for 5xx. In ACME mode certmagic's certificate and renewal messages are written to stdout as JSON in the same shape (`time`, `level`, `msg`). Use `docker compose logs -f` to follow them.

### Audit log

Every change, login, failed login, failed password change, report download, backup download and `reset-password` run is also written to stdout as an `audit` JSON line (`username`, `action`, `target`, `details`, `ip`, and `user_id` when known) and stored in the database, where admins browse it at `/admin/audit` (paged, filterable by user, action and date). Counts entered and locks made by a paired device have the username `device`; the `reset-password` command uses `cli`. Repeated failed logins from one client for the same username are coalesced into one audit entry per time window. The log is append-only and kept forever, so it grows with use; it contains usernames and client IP addresses, so apply your club's retention policy to the stdout lines. See `docs/spec.md`.

### Score log

Each time a stage changes status (locked, accepted, disputed, unlocked, or edited by the admin), a `stage_status` line is also written to stdout with the competition, entry, shooter, ring counts, the stage status and who locked or accepted it (see `docs/spec.md`). Keep your container logs: if the database is lost and there is no replica, they are the only record of the final scores. The lines contain personal data (WAID, names, club), so apply your club's retention policy to them.

To preview what a log would restore, run `matchmark restore [file]` (or pipe the log on stdin, e.g. `docker compose logs --no-log-prefix matchmark | matchmark restore`). It needs no `ADMIN_USERNAME` or `ADMIN_PASSWORD`, skips non-score and malformed lines with a count, rejects any `schema_version` other than 2 (so logs written before the stage status model must be restored with the earlier release), treats lines with an unknown course key or status, an invalid `locked_by` or an `emitted_at` outside the fixed-width layout as malformed, and prints the competitions, entries and scored stages it would restore. Without `--apply` it is a dry run and writes nothing.

To write the restore, run `matchmark restore --apply [file]` with the same `DB_PATH` the server uses (for example `docker compose run --rm -T matchmark restore --apply < scores.jsonl` while the server is stopped). It opens the database, runs migrations, and recreates competitions (as `finished` when every stage is accepted, otherwise `active` with the current stage at the lowest stage that is not fully accepted), shooters, entries and the stages with their ring counts, status, `locked_by` lane (or username), `scorer_name`, `locked_at` and the accepted fields, keeping the original competition and entry ids. Flags may come before or after the file. If the database already contains competitions or shooters, it refuses and changes nothing; the whole restore is one transaction, so a failure leaves the database unchanged; add `--merge` to apply only data whose ids match or are new and list conflicts (differing competition, entry, or non-open stage counts, `locked_by`, `scorer_name` or `locked_at`) without overwriting them. With `--merge` it is idempotent by ids: re-running changes nothing, and stages that are not open are left untouched. The log has no sex or country, so restored shooters get sex `M` and an empty country. Unscored entries, devices and pairings are not restored; the command prints a note saying so and that the devices must be re-paired before scoring continues.

## Lockout recovery

If every admin is locked out, stop the server and run `matchmark reset-password <username>` with the same `DB_PATH` the server uses, passing the new temporary password on stdin (so it stays out of the process list), for example `printf '%s\n' "$NEW_PASSWORD" | docker compose run --rm -T matchmark reset-password alice`. The username is case-insensitive. It needs no `ADMIN_USERNAME` or `ADMIN_PASSWORD`, rejects an unknown username, a missing database and a password outside 16-128 characters without changing anything, and on success sets the new password, ends all of that user's sessions and flags the account so the password must be changed at next login. The run is audited (actor `cli`, an `audit` line on stdout).

## TLS modes

The session cookie is `Secure` when the binary serves TLS itself (static or ACME mode) or when `BASE_URL` is an `https` URL; otherwise (plain HTTP without an `https` `BASE_URL`) it is not.

- **Plain HTTP (default):** for local networks or behind a TLS-terminating reverse proxy. Set `BASE_URL` to the public URL and `TRUSTED_PROXY` to the proxy address.
- **Static certificate:** mount the files and set `TLS_CERT_FILE` and `TLS_KEY_FILE`; the server listens on `:8443`. Map the port in compose, e.g. `"443:8443"`.
- **ACME:** set `ACME_DOMAIN` and `ACME_EMAIL`. The server obtains and renews certificates itself using the HTTP-01 challenge, which needs the public port 80 forwarded to the container's `:8080`, with HTTPS served on `:8443` (map `"80:8080"` and `"443:8443"`). Certificates are stored under `/data/certs`. Set `BASE_URL` to `https://<domain>`. The default compose file only publishes `8080` (as `HTTP_PORT`), so in ACME mode replace its `ports` entry with:

  ```yaml
  ports:
    - "80:8080"
    - "443:8443"
  ```

  If public port 80 does not reach `:8080`, certificate issuance and renewal fail.

### Live updates

Spectator, admin and scorer screens receive live updates over server-sent events (`text/event-stream`): `GET /api/spectator/competitions/{id}/stream`, `GET /api/competitions/{id}/stream` and `GET /api/scorer/stream`. A reverse proxy must not buffer or cache these responses and must allow them to stay open: the server sends `X-Accel-Buffering: no` and `Cache-Control: no-cache`, a heartbeat every 25 seconds, and closes each stream after 30 minutes, so keep the proxy read timeout above 25 seconds (nginx: `proxy_buffering off`, `proxy_read_timeout 1h`). Each role has its own cap on concurrent streams: 500 spectator, 50 admin and 100 scorer streams, so a spectator flood cannot block admin or scorer screens. Requests over a cap get `503` with `Retry-After`; requests for unknown or draft competitions are rejected before subscribing.

## Pairing flow

1. Admin creates a competition, adds entries and assigns lanes.
2. Admin opens pairing and shows or shares the pairing link (QR code) for each lane; `BASE_URL` must be set so links point at the public address.
3. A scorer opens the link once on a phone. The device is paired to that lane and thereafter sees only the lane it scores (the next manned lane, wrapping).
4. Admin can revoke a pairing at any time.

## Backup

The admin UI offers a database backup download to admins who have been allowed to by another admin on the Users screen (the seeded admin is allowed; other admins and regular users get 403 `backup_not_permitted`) (SQLite online backup, safe while running). Alternatively stop the container and copy `data/matchmark.db`. Restore by placing the file at `data/matchmark.db` before starting.

## Replication

Optional and off by default. With `REPLICATION_URL` set, the database is opened through the Turso sync API and pushed to a self-hosted `tursodb --sync-server` every `REPLICATION_INTERVAL` and once when a competition is finished. Push-only: nothing is pulled back. A failed push is logged and retried on the next tick and never blocks scoring. Without `REPLICATION_URL` the database opens through plain `database/sql`; both modes run the same migrations.

Run the sync server on a second host (the replica's database file is `/data/replica.db`):

```yaml
services:
  replica:
    image: ghcr.io/tursodatabase/turso:latest
    command: ["tursodb", "/data/replica.db", "--sync-server", "0.0.0.0:8080"]
    volumes:
      - ./replica-data:/data
    ports:
      - "10.0.0.2:8080:8080"
```

Adjust the image or binary to your install of `tursodb`, and bind the port to the private address only. On the primary set `REPLICATION_URL=http://10.0.0.2:8080`. For local testing, `brew install turso` provides `tursodb`; run `tursodb replica.db --sync-server 127.0.0.1:8080` and point `REPLICATION_URL` at `http://127.0.0.1:8080`.

> **Warning:** the sync server has no authentication. Expose it only on a private network (WireGuard, Tailscale or LAN), never on the public internet. Where the replica must cross the public internet, copy snapshot files instead.

Restore by copying the replica's database file to `data/matchmark.db` on a new primary.

## Targets

| Target | Use |
|--------|-----|
| linux/amd64 | Supported (Docker image and release binary) |
| linux/arm64 | Supported, including 64-bit Raspberry Pi OS (Docker image and release binary) |
| darwin/amd64, darwin/arm64 | Release binary, unsigned |
| windows/amd64 | Release binary, unsigned |

32-bit ARM and windows/arm64 are unsupported: tursogo ships no native library for them. The binary is built with `CGO_ENABLED=0` but loads an embedded native library at runtime, which the final image extracts to `TURSO_GO_CACHE_DIR` (`/var/cache/turso`, writable by the non-root user). On Alpine the build needs `-tags musl`, and the image installs `libgcc`.

On a Pi (64-bit OS) building on the device works unchanged:

```sh
docker compose up --build -d
```

To cross-build from another machine: `docker buildx build --platform linux/arm64 -t matchmark:latest .`.

## Releases

Push a tag such as `v0.1.0` (or `v0.1.0-rc.1` for a pre-release) on a commit on `main`. The release workflow runs all tests, then pushes `ghcr.io/sollie/matchmark:v0.1.0`, `:v0.1` and `:latest` (multi-arch, amd64 and arm64; pre-releases only get the exact tag), then publishes a GitHub Release with archives named `matchmark_<version>_<os>_<arch>` (`.tar.gz`, `.zip` on Windows) and `checksums.txt`. The changelog comes from Conventional Commits. `matchmark --version` prints the version. Restrict who may push `v*` tags with a tag ruleset in the repository settings (manual).

The macOS and Windows binaries are not signed. On macOS clear the quarantine flag (`xattr -d com.apple.quarantine matchmark`); on Windows choose "More info" and "Run anyway" in SmartScreen. See ADR 0012.

## CI

Pull requests run the source checks (build, windows and darwin cross-compile, test, lint), the web checks and chromium e2e. The image build and the goreleaser snapshot run on a PR only when the Dockerfile, smoke script, `go.mod`/`go.sum`, `.goreleaser.yaml` or the workflows change. On `main` and weekly the image is built on native runners (`ubuntu-latest`, `ubuntu-24.04-arm`), smoke-tested with `scripts/smoke-image.sh matchmark:latest` and scanned with Trivy, and all three browsers run e2e.

## License

MatchMark is released under the [MIT License](LICENSE).
