# MatchMark

WA1500/PPC competition scoring system: one Go binary (REST API + embedded Svelte SPA) backed by a single SQLite file. See [`docs/spec.md`](docs/spec.md) and [`docs/v2-plan.md`](docs/v2-plan.md).

The rulebook (courses, weapon classes, ring values) is public at `GET /api/rulebook`; the SPA reads its course labels from it.

## Layout

| Path | Description |
|------|-------------|
| `source/` | Go module (`cmd/server`, `internal/`, `api/openapi.yaml`) |
| `source/web/` | Svelte + Vite + TypeScript SPA, built to `source/web/dist` and embedded in the binary |
| `Dockerfile` | Node build stage, Go build stage, Alpine 3.24 final image built with `-tags musl` (non-root) |
| `docker-compose.yml` | The app plus a `./data` volume |
| `scripts/smoke-image.sh` | Starts an image and checks it opens its database |
| `docs/` | Spec, ADRs, plan |

## Run with Docker

```sh
mkdir -p data && sudo chown 65532:65532 data
export ADMIN_PASSWORD='at-least-16-characters'
docker compose up --build -d
```

The app listens on plain HTTP at `http://localhost:8080` (override the host port with `HTTP_PORT`). The container runs as UID/GID 65532, so the `data` directory must be writable by that user. The database (a SQLite-format file opened through the tursogo driver), backups and ACME certificates live in `/data`.

Sign in at `/` with the admin password. `ADMIN_PASSWORD` must be at least 16 characters and is the only required setting.

## Run without Docker

Prerequisites: Go 1.27+ and Node.js 24 LTS.

```sh
cd source/web && npm ci && npm run build && cd ..
ADMIN_PASSWORD='at-least-16-characters' CGO_ENABLED=0 go run ./cmd/server
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

Set `E2E_BASE_URL` (and `E2E_ADMIN_PASSWORD`) to test an already running instance instead of starting one. CI runs one job per browser in parallel on every pull request, gated by an aggregate `E2E` check, and uploads a per-browser report on failure.

Coverage, by area:

- Auth: wrong and correct password, per-client login lockout, sign out, 401 on unauthenticated admin API calls, the not-found page, and the unpaired scorer page.
- Shooters: add, edit, cancel an edit, delete with confirmation, and form validation.
- Competitions: creation for each course of fire with its stage columns, the prefilled ISO 8601 date and invalid-date blocking, the list with statuses, start and finish with confirmation, finished competitions (revoked pairings, no new entries), editing a draft (name, date, course, locked course once entries exist, no edit once started), delete with confirmation, and visibility flags persisting.
- Entries and admin scoring: adding entries with category and default lane, taken-lane rejection, move, withdraw and remove, live shot totals with rejection of wrong totals, and save and lock as "Admin" with unlock.
- Pairing and scorer device: pairing link creation, the device's lane and score lane, score lane wrap-around, running total and mismatch message, save and lock, read-only locked stages, revoked pairings, the admin-scoring requirement with one manned lane, unpairing a device, and refusal to score a draft competition.
- Spectator: ranking by total then X count with stage progress, draft and finished visibility, each visibility flag, the empty state, and 404 for a draft standings API.
- Reports and backup: `.xlsx` results report, detailed CSV, database backup, and 401 without a session.

Not covered by the browser suite: the score log (Go tests, ADR 0007), accessible-name cleanup (#129), and phone and tablet viewports.

Tests seed data through the admin API with unique names and send a unique `X-Forwarded-For` per client so each one has its own login rate limit. The e2e server trusts `127.0.0.1`, and `docker-compose.e2e.yml` sets `TRUSTED_PROXY` to the private network ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), so the e2e container's peer address is trusted and its forwarded headers are honoured.


## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_PASSWORD` | required | Admin password, minimum 16 characters |
| `DB_PATH` | `data/matchmark.db` (`/data/matchmark.db` in the image) | SQLite file; its directory also holds backups and ACME certificates |
| `LISTEN_ADDR` | `:8080` plain, `:8443` with TLS | Listen address |
| `BASE_URL` | empty | Public URL, used for pairing links; an `https` URL marks the session cookie `Secure` |
| `TRUSTED_PROXY` | empty | Comma-separated IPs or CIDRs of reverse proxies whose forwarded client IP headers are trusted |
| `DISABLE_COMPRESSION` | `false` | Set to `true` to turn off response compression (for example behind a proxy that already compresses) |
| `TLS_CERT_FILE`, `TLS_KEY_FILE` | empty | Static certificate mode; set both |
| `ACME_DOMAIN`, `ACME_EMAIL` | empty | Automatic certificates (Let's Encrypt); both required together |
| `REPLICATION_URL` | empty | Base URL of a `tursodb --sync-server`; empty disables [Replication](#replication) |
| `REPLICATION_INTERVAL` | `60s` | Push interval as a Go duration (for example `30s`, `5m`); must be positive |

Static and ACME modes are mutually exclusive.

## Logging

Text responses (SPA assets, JSON, CSV) of at least 256 bytes are compressed with Brotli, gzip or deflate according to the request `Accept-Encoding` (preference: `br`, `gzip`, `deflate`). `.xlsx` reports, other binary types, HEAD and Range requests are left alone. Browsers only offer Brotli over HTTPS.

The server writes structured JSON logs (`log/slog`) to stdout: one `request` line per HTTP request with `method`, `path` (no query string), `status`, `bytes`, `duration_ms`, `remote` (the client IP, honouring `TRUSTED_PROXY`) and `user_agent`. Level is `INFO` for 1xx-3xx, `WARN` for 4xx and `ERROR` for 5xx. In ACME mode certmagic's certificate and renewal messages are written to stdout as JSON in the same shape (`time`, `level`, `msg`). Use `docker compose logs -f` to follow them.

### Score log

Each time a stage is locked, unlocked, or edited by the admin while locked, a `stage_locked` or `stage_unlocked` line is also written to stdout with the competition, entry, shooter, ring counts and who locked it (see `docs/spec.md`). Keep your container logs: if the database is lost and there is no replica, they are the only record of the final scores. The lines contain personal data (WAID, names, club), so apply your club's retention policy to them.

To preview what a log would restore, run `matchmark restore [file]` (or pipe the log on stdin, e.g. `docker compose logs --no-log-prefix matchmark | matchmark restore`). It needs no `ADMIN_PASSWORD`, skips non-score and malformed lines with a count, rejects an unknown `schema_version`, treats lines with an unknown course key, an invalid `locked_by` or an `emitted_at` outside the fixed-width layout as malformed, and prints the competitions, entries and scored stages it would restore. Without `--apply` it is a dry run and writes nothing.

To write the restore, run `matchmark restore --apply [file]` with the same `DB_PATH` the server uses (for example `docker compose run --rm -T matchmark restore --apply < scores.jsonl` while the server is stopped). It opens the database, runs migrations, and recreates competitions (as `finished`), shooters, entries and the locked stages with their ring counts, `locked_by` lane (or `Admin`), `scorer_name` and `locked_at`, keeping the original competition and entry ids. Flags may come before or after the file. If the database already contains competitions or shooters, it refuses and changes nothing; the whole restore is one transaction, so a failure leaves the database unchanged; add `--merge` to apply only data whose ids match or are new and list conflicts (differing competition, entry, or locked stage counts, `locked_by`, `scorer_name` or `locked_at`) without overwriting them. With `--merge` it is idempotent by ids: re-running changes nothing, and stages already locked are left untouched. The log has no sex or country, so restored shooters get sex `M` and an empty country. Unscored entries, devices and pairings are not restored; the command prints a note saying so.

## TLS modes

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

Spectator, admin and scorer screens receive live updates over server-sent events (`text/event-stream`): `GET /api/spectator/competitions/{id}/stream`, `GET /api/competitions/{id}/stream` and `GET /api/scorer/stream`. A reverse proxy must not buffer or cache these responses and must allow them to stay open: the server sends `X-Accel-Buffering: no` and `Cache-Control: no-cache`, a heartbeat every 25 seconds, and closes each stream after 30 minutes, so keep the proxy read timeout above 25 seconds (nginx: `proxy_buffering off`, `proxy_read_timeout 1h`). At most 500 streams are served at once.

## Pairing flow

1. Admin creates a competition, adds entries and assigns lanes.
2. Admin opens pairing and shows or shares the pairing link (QR code) for each lane; `BASE_URL` must be set so links point at the public address.
3. A scorer opens the link once on a phone. The device is paired to that lane and thereafter sees only the lane it scores (the next manned lane, wrapping).
4. Admin can revoke a pairing at any time.

## Backup

The admin UI offers a database backup download (SQLite online backup, safe while running). Alternatively stop the container and copy `data/matchmark.db`. Restore by placing the file at `data/matchmark.db` before starting.

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
| linux/amd64 | Supported (Docker image) |
| linux/arm64 | Supported, including 64-bit Raspberry Pi OS (Docker image) |
| darwin/arm64 | Development only |

32-bit ARM is unsupported: tursogo ships no native library for it. The binary is built with `CGO_ENABLED=0` but loads an embedded native library at runtime, which the final image extracts to `TURSO_GO_CACHE_DIR` (`/var/cache/turso`, writable by the non-root user). On Alpine the build needs `-tags musl`, and the image installs `libgcc`.

On a Pi (64-bit OS) building on the device works unchanged:

```sh
docker compose up --build -d
```

To cross-build from another machine: `docker buildx build --platform linux/arm64 -t matchmark:latest .`.

CI builds, tests and smoke-tests the image on native runners (`ubuntu-latest`, `ubuntu-24.04-arm`) and builds darwin/arm64. Run the smoke test locally with `scripts/smoke-image.sh matchmark:latest`.

## License

MatchMark is released under the [MIT License](LICENSE).
