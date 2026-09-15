FROM node:26.8-alpine3.23 AS frontend-builder

WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

FROM caddy:2.11.1-alpine

COPY --from=frontend-builder /app/build /srv/frontend
COPY caddy/Caddyfile /etc/caddy/Caddyfile
