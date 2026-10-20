---
status: accepted
---

# Response compression in the server binary

The server compresses eligible responses itself with one middleware inside the request logger, so logged `bytes` are the bytes sent. It negotiates `Accept-Encoding` with q-values and prefers `br`, then `gzip`, then `deflate`. Only text-like content types of at least 256 bytes (when the length is known) are compressed; `.xlsx`, HEAD, Range, 204/304 and already-encoded responses are not. `Vary: Accept-Encoding` is always added, `Content-Length` is dropped and strong ETags are weakened. `DISABLE_COMPRESSION=true` turns it off.

gzip and deflate use the standard library. Brotli uses `github.com/andybalholm/brotli`, which is pure Go (so `CGO_ENABLED=0` holds) and actively released. A packaged middleware (`CAFxX/httpcompression`) was rejected because its last release is from 2023.

## Consequences

Compressed bodies that mix secrets with attacker-controlled input are open to BREACH-style attacks. API responses carry no CSRF tokens or reflected input, and the session is a cookie, so the risk is accepted. zstd and build-time precompression of the SPA are not done.
