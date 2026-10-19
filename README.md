# MatchMark

WA1500/PPC competition scoring system: one Go binary (REST API + embedded Svelte SPA) backed by a single SQLite file. See [`docs/spec.md`](docs/spec.md) and [`docs/v2-plan.md`](docs/v2-plan.md).

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

Set `E2E_BASE_URL` (and `E2E_ADMIN_PASSWORD`) to test an already running instance instead of starting one. CI runs the suite on every pull request and uploads the report on failure.

Current coverage is a smoke test: admin sign-in and competition creation. Planned expansion:

- Shooter management and entries
- Device pairing and scorer stage entry
- Score lane, locks and admin unlock
- Spectator views and visibility flags
- Reports (`.xlsx` and CSV downloads) and backup
- Phone and tablet viewport projects


## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_PASSWORD` | required | Admin password, minimum 16 characters |
| `DB_PATH` | `data/matchmark.db` (`/data/matchmark.db` in the image) | SQLite file; its directory also holds backups and ACME certificates |
| `LISTEN_ADDR` | `:8080` plain, `:8443` with TLS | Listen address |
| `BASE_URL` | empty | Public URL, used for pairing links; an `https` URL marks the session cookie `Secure` |
| `TRUSTED_PROXY` | empty | Comma-separated IPs or CIDRs of reverse proxies whose forwarded client IP headers are trusted |
| `TLS_CERT_FILE`, `TLS_KEY_FILE` | empty | Static certificate mode; set both |
| `ACME_DOMAIN`, `ACME_EMAIL` | empty | Automatic certificates (Let's Encrypt); both required together |
| `REPLICATION_URL` | empty | Base URL of a `tursodb --sync-server`; empty disables [Replication](#replication) |
| `REPLICATION_INTERVAL` | `60s` | Push interval as a Go duration (for example `30s`, `5m`); must be positive |

Static and ACME modes are mutually exclusive.

## Logging

The server writes structured JSON logs (`log/slog`) to stdout: one `request` line per HTTP request with `method`, `path` (no query string), `status`, `bytes`, `duration_ms`, `remote` (the client IP, honouring `TRUSTED_PROXY`) and `user_agent`. Level is `INFO` for 1xx-3xx, `WARN` for 4xx and `ERROR` for 5xx. In ACME mode certmagic's certificate and renewal messages are written to stdout as JSON in the same shape (`time`, `level`, `msg`). Use `docker compose logs -f` to follow them.

### Score log

Each time a stage is locked, unlocked, or edited by the admin while locked, a `stage_locked` or `stage_unlocked` line is also written to stdout with the competition, entry, shooter, ring counts and who locked it (see `docs/spec.md`). Keep your container logs: if the database is lost and there is no replica, they are the only record of the final scores. The lines contain personal data (WAID, names, club), so apply your club's retention policy to them.

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
