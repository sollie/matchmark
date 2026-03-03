# MatchMark

WA1500/PPC competition scoring system.

## Components

| Directory | Description |
|-----------|-------------|
| `backend/` | Go REST API — chi router, SQLite, JWT auth |
| `frontend/` | SvelteKit PWA — offline-first, touch-optimized |
| `caddy/` | Caddyfile — TLS termination, reverse proxy, static files |
| `docker-compose.yml` | Orchestrates all three services |
| `docs/plan/` | Full project specification |

## Quick Start

Copy `.env.example` to `.env` and fill in the required values, then:

```sh
docker compose up --build
```

The app will be available at the domain configured in `DOMAIN`.

## Local Development

### With Docker

Use the dev Compose override (plain HTTP, no TLS):

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

This swaps in `caddy/Caddyfile.dev` (plain HTTP on port 80) and sets `FRONTEND_ORIGIN` to empty. The app is available at `http://localhost`.

### Without Docker

Prerequisites: [Go](https://go.dev/dl/), [Node.js](https://nodejs.org/), [Caddy](https://caddyserver.com/docs/install).

**1. Build the frontend:**

```sh
cd frontend
npm install
npm run build
cd ..
```

**2. Start the backend:**

```sh
export JWT_SECRET=dev-secret-change-me-at-least-32-chars
export ADMIN_PASSWORD=dev-admin-password
export DB_PATH=./matchmark.db
go run ./cmd/server
```

The API listens on `:8080`.

**3. Start Caddy:**

```sh
caddy run --config caddy/Caddyfile.local
```

Caddy serves `https://localhost` (auto-TLS via its local CA — trust the root cert once when prompted). It proxies `/api/*` to `:8080` and serves the built frontend from `frontend/build`.

The app is available at `https://localhost`. Re-run `npm run build` in the `frontend/` directory after any frontend changes.

## Environment Variables

See `.env.example` for the full list. Required variables:

| Variable | Description |
|----------|-------------|
| `DOMAIN` | Domain name for Caddy TLS (e.g. `scores.example.com`) |
| `JWT_SECRET` | Random secret, minimum 32 bytes |
| `ADMIN_PASSWORD` | Initial admin password, minimum 16 characters |

Optional variables:

| Variable | Description |
|----------|-------------|
| `FRONTEND_ORIGIN` | Restrict CORS to this origin (e.g. `https://scores.example.com`). Leave empty when the frontend is served from the same origin as the API (normal Caddy setup). |

## Documentation

See [`docs/plan/00-index.md`](docs/plan/00-index.md) for the full specification index.
