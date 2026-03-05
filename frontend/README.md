# MatchMark Frontend

SvelteKit PWA for the MatchMark WA1500/PPC competition scoring system.

Built with SvelteKit (`adapter-static`), TypeScript, and `@vite-pwa/sveltekit`.

## Prerequisites

- Node.js 20+
- npm

## Setup

```sh
npm install
```

## Development

```sh
npm run dev
```

The dev server starts at `http://localhost:5173`. It expects the backend API to be available at `/api` — use the Caddy reverse proxy or configure Vite's proxy in `vite.config.ts` for local development.

## Building

```sh
npm run build
```

Produces a fully static bundle in `build/`. This is served by Caddy in production (see `caddy/Dockerfile`).

## Type checking

```sh
npm run check
```

## Production deployment

The frontend is built inside the Caddy Docker image — see `caddy/Dockerfile` in the repo root. No separate frontend service is needed.

For full stack local development, use Docker Compose from the repo root:

```sh
docker compose up
```
