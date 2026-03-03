# MatchMark — Architecture Overview

## System Components

```
                  ┌──────────────────────────────────┐
  Browser/PWA ───▶│  Caddy (TLS + static + proxy)    │
                  │  example.com  :80/:443            │
                  └──────┬───────────────┬────────────┘
                         │ /api/*        │ everything else
                         ▼               ▼
                  ┌────────────┐  ┌─────────────────┐
                  │  Backend   │  │  Static frontend │
                  │  Go :8080  │  │  /srv/frontend   │
                  │  chi router│  │  (SvelteKit SPA) │
                  └──────┬─────┘  └─────────────────┘
                         │
                  ┌──────▼─────┐
                  │  SQLite DB │
                  │  /data/    │
                  │  matchmark │
                  │  .db       │
                  └────────────┘
```

## Docker Compose Services

| Service | Image | Role |
|---------|-------|------|
| `backend` | custom Go build | REST API, business logic, SQLite |
| `frontend` | custom Node build | Builds SvelteKit; copies static output to shared volume then exits |
| `caddy` | `caddy:2-alpine` | TLS (Let's Encrypt), static file server, reverse proxy |

## Named Volumes

| Volume | Used by | Contents |
|--------|---------|----------|
| `db_data` | backend | `matchmark.db` |
| `frontend_dist` | frontend (write), caddy (read) | Compiled SvelteKit output |
| `caddy_data` | caddy | ACME certs, Caddy state |
| `caddy_config` | caddy | Caddy runtime config |

## Repository Layout

```
matchmark/
├── backend/
│   ├── cmd/server/
│   │   └── main.go                  # entrypoint — wires DI, starts HTTP server
│   ├── internal/
│   │   ├── auth/
│   │   │   ├── jwt.go               # sign / verify / refresh tokens
│   │   │   ├── password.go          # argon2id hash / verify
│   │   │   └── middleware.go        # chi middleware: RequireAuth, RequireAdmin
│   │   ├── db/
│   │   │   ├── migrate.go           # embedded SQL migration runner
│   │   │   ├── seed.go              # admin user seed (reads ADMIN_PASSWORD env)
│   │   │   └── migrations/
│   │   │       └── 001_init.sql     # full schema (see 02-database.md)
│   │   ├── model/
│   │   │   └── model.go             # shared Go structs
│   │   ├── rulebook/
│   │   │   ├── courses.go           # stage definitions per discipline
│   │   │   └── weapons.go           # allowed weapon classes per discipline
│   │   ├── repository/
│   │   │   ├── user.go
│   │   │   ├── competition.go
│   │   │   ├── shooter.go
│   │   │   ├── stage.go
│   │   │   ├── hit.go
│   │   │   └── lock.go
│   │   ├── service/
│   │   │   ├── auth.go
│   │   │   ├── competition.go
│   │   │   ├── shooter.go
│   │   │   ├── scoring.go           # hit submission, lane derivation, lock logic
│   │   │   └── monitor.go           # SSE data aggregation
│   │   ├── handler/
│   │   │   ├── auth.go
│   │   │   ├── user.go
│   │   │   ├── competition.go
│   │   │   ├── shooter.go
│   │   │   ├── scoring.go
│   │   │   ├── monitor.go           # SSE endpoint
│   │   │   └── export.go
│   │   └── export/
│   │       ├── pdf.go               # gofpdf scorecard layout
│   │       ├── csv.go
│   │       └── json.go
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── app.html
│   │   ├── app.css
│   │   ├── lib/
│   │   │   ├── api/
│   │   │   │   ├── client.ts        # fetch wrapper with JWT + silent refresh
│   │   │   │   ├── auth.ts
│   │   │   │   ├── competitions.ts
│   │   │   │   ├── scoring.ts
│   │   │   │   └── admin.ts
│   │   │   ├── stores/
│   │   │   │   ├── auth.ts          # JWT, user role, username
│   │   │   │   ├── session.ts       # active competitionId, ownLane, scoreLane
│   │   │   │   └── scoring.ts       # stage data, hits, lock state
│   │   │   ├── db/
│   │   │   │   └── idb.ts           # IndexedDB helper (scoring queue)
│   │   │   └── sync/
│   │   │       └── queue.ts         # queue hits, replay on reconnect
│   │   ├── routes/
│   │   │   ├── +layout.svelte       # auth guard + nav
│   │   │   ├── login/
│   │   │   │   └── +page.svelte
│   │   │   ├── score/
│   │   │   │   ├── select/
│   │   │   │   │   └── +page.svelte # pick competition + own lane
│   │   │   │   ├── stage/
│   │   │   │   │   └── +page.svelte # scoring keypad (main UI)
│   │   │   │   └── summary/
│   │   │   │       └── +page.svelte # read-only summary after all locked
│   │   │   └── admin/
│   │   │       ├── +layout.svelte   # admin-only guard
│   │   │       ├── +page.svelte     # admin dashboard
│   │   │       ├── users/
│   │   │       │   └── +page.svelte
│   │   │       └── competitions/
│   │   │           ├── +page.svelte
│   │   │           ├── [id]/
│   │   │           │   ├── +page.svelte      # competition detail + SSE monitor
│   │   │           │   └── export/
│   │   │           │       └── +page.svelte
│   │   │           └── new/
│   │   │               └── +page.svelte
│   │   └── service-worker.ts
│   ├── static/
│   │   ├── manifest.webmanifest
│   │   └── icons/                   # PWA icons (192×192, 512×512, maskable)
│   ├── svelte.config.js
│   ├── vite.config.ts
│   ├── package.json
│   └── Dockerfile
├── caddy/
│   └── Caddyfile
├── docker-compose.yml
├── .env.example
├── AGENTS.md
└── docs/
    └── plan/                        # this directory
```

## Caddy Configuration

```caddyfile
{$DOMAIN} {
    encode gzip

    root * /srv/frontend
    file_server

    handle /api/* {
        reverse_proxy backend:8080 {
            header_up Host {host}
            header_up X-Forwarded-For {remote_host}
        }
    }

    # SPA fallback: serve index.html for all non-file paths
    handle {
        try_files {path} /index.html
        file_server
    }
}
```

The `DOMAIN` environment variable is passed to the `caddy` service in `docker-compose.yml`
and substituted at runtime using Caddy's native `{$VAR}` syntax — no image rebuild needed
when changing the domain.
```
