# MatchMark

WA1500/PPC competition scoring system: one Go binary (REST API + embedded Svelte SPA) backed by a single SQLite file. See [`docs/spec.md`](docs/spec.md) and [`docs/v2-plan.md`](docs/v2-plan.md).

## Layout

| Path | Description |
|------|-------------|
| `source/` | Go module (`cmd/server`, `internal/`, `api/openapi.yaml`) |
| `source/web/` | Svelte + Vite + TypeScript SPA, built to `source/web/dist` and embedded in the binary |
| `Dockerfile` | Node build stage, Go build stage, `scratch` final image (CA certificates, non-root) |
| `docker-compose.yml` | The app plus a `./data` volume |
| `docs/` | Spec, ADRs, plan |

## Run with Docker

```sh
mkdir -p data && sudo chown 65532:65532 data
export ADMIN_PASSWORD='at-least-16-characters'
docker compose up --build -d
```

The app listens on plain HTTP at `http://localhost:8080` (override the host port with `HTTP_PORT`). The container runs as UID/GID 65532, so the `data` directory must be writable by that user. The SQLite database, backups and ACME certificates live in `/data`.

Sign in at `/` with the admin password. `ADMIN_PASSWORD` must be at least 16 characters and is the only required setting.

## Run without Docker

Prerequisites: Go 1.26+ and Node.js 24 LTS.

```sh
cd source/web && npm ci && npm run build && cd ..
ADMIN_PASSWORD='at-least-16-characters' CGO_ENABLED=0 go run ./cmd/server
```

For frontend development run `npm run dev` in `source/web`; it proxies `/api` to `localhost:8080`.

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

Adjust the image or binary to your install of `tursodb`, and bind the port to the private address only. On the primary set `REPLICATION_URL=http://10.0.0.2:8080`.

> **Warning:** the sync server has no authentication. Expose it only on a private network (WireGuard, Tailscale or LAN), never on the public internet. Where the replica must cross the public internet, copy snapshot files instead.

Restore by copying the replica's database file to `data/matchmark.db` on a new primary.

## Raspberry Pi / arm64

The binary is pure Go (`CGO_ENABLED=0`) and the base images are multi-arch, so building on the device works unchanged:

```sh
docker compose up --build -d
```

To cross-build from another machine: `docker buildx build --platform linux/arm64 -t matchmark:latest .`. The first build on a Pi is slow (npm install); the resulting image is about 20 MB.
