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

## Local Security Scanning

### CodeQL

Requires the [CodeQL CLI](https://docs.github.com/en/code-security/codeql-cli/getting-started-with-the-codeql-cli/setting-up-the-codeql-cli). On macOS:

```sh
brew install codeql
```

Run both the Go and JavaScript/TypeScript scans from the repo root:

```sh
./scripts/codeql-local.sh
```

The script downloads the required query packs on first run (cached under `~/.codeql/packages` for subsequent runs), creates CodeQL databases for `backend/` (Go) and `frontend/` (JavaScript/TypeScript), analyzes them with the `security-and-quality` query suite, and prints a summary table to the console. It exits non-zero if any findings are present.

Output files written to the repo root (gitignored):

| File | Description |
|------|-------------|
| `results-go.sarif` / `results-js.sarif` | Full SARIF — open in the VS Code SARIF Viewer or upload to the GitHub Security tab |
| `results-go.csv` / `results-js.csv` | CSV with columns: name, description, severity, message, path, start\_line, start\_col, end\_line, end\_col |

SARIF files can be viewed with the [SARIF Viewer](https://marketplace.visualstudio.com/items?itemName=MS-SarifVSCode.sarif-viewer) extension in VS Code, or uploaded to the GitHub Security tab.

## Documentation

See [`docs/plan/00-index.md`](docs/plan/00-index.md) for the full specification index.
