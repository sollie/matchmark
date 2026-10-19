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
