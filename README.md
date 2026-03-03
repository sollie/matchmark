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

## Environment Variables

See `.env.example` for the full list. Required variables:

| Variable | Description |
|----------|-------------|
| `DOMAIN` | Domain name for Caddy TLS (e.g. `scores.example.com`) |
| `JWT_SECRET` | Random secret, minimum 32 bytes |
| `ADMIN_PASSWORD` | Initial admin password, minimum 16 characters |

## Documentation

See [`docs/plan/00-index.md`](docs/plan/00-index.md) for the full specification index.
