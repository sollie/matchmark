---
status: accepted
---

# Turso Database via tursogo, Alpine image, optional Replication

MatchMark uses `turso.tech/database/tursogo` instead of `modernc.org/sqlite`, to get Replication of the club's data to a self-hosted second host for safekeeping. Turso Database is a pre-1.0 Rust rewrite of SQLite that reads the SQLite file format; we accept that maturity risk and rely on host-level snapshots as the real safety net.

tursogo is built with `CGO_ENABLED=0` but loads an embedded native library with `dlopen` at runtime. This supersedes the `scratch` final image in ADR 0003: the final image is Alpine 3.24 (latest stable) built with `-tags musl`, running non-root with `TURSO_GO_CACHE_DIR` pointing at a writable path, because the library is extracted there on first use.

## Decisions

- Supported targets: linux/amd64 and linux/arm64 (64-bit Raspberry Pi OS only); darwin/arm64 for development. 32-bit ARM is unsupported because upstream ships no library for it.
- Replication is push-only, optional, and off by default. When configured the primary opens through the tursogo sync API, otherwise through plain `database/sql`; both run the same migrations. It runs on a timer (default 60s) and once when a competition finishes; failures are logged and retried and never block scoring.
- The replica is `tursodb --sync-server` on a second host. It has no authentication and is documented upstream as for local and development use, so it is reachable only over a private network (WireGuard, Tailscale or LAN). Where the replica must cross the public internet, ship snapshot files instead.
- Restore is copying the replica's database file to a new primary. No drill is specified.
- CI builds, tests and smoke-tests the Docker image on native runners (`ubuntu-latest` for amd64, `ubuntu-24.04-arm` for arm64, no QEMU), plus a build-only darwin/arm64 job. The `dlopen` failure mode only shows at runtime, so the smoke test opens a database inside each image.
- Before any other work, a spike runs the existing migration and full test suite against tursogo and checks `foreign_keys`, `busy_timeout`, `SetMaxOpenConns(1)` and the admin backup download, which assumed SQLite's online backup.

## Considered Options

- Keep `modernc.org/sqlite` and back up with file copies: rejected because it gives no incremental Replication and the project wants to adopt Turso.
- Turso Cloud as the remote: rejected to avoid paid hosting.
- Client-side database sync from devices: rejected, the offline-first scorer flow was removed in v2 and the server stays the only writer.

## Consequences

The image is larger and no longer a single static binary. Anything using SQLite-specific features (online backup, extensions) must be checked against Turso's compatibility notes.
