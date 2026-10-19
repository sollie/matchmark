# One Go binary with embedded frontend, shipped as a scratch image

The app must run anywhere a Docker daemon runs, including a Raspberry Pi, so there is no bundled proxy and no separate frontend service. The Svelte SPA is built in an Alpine Node stage, embedded into the Go binary built in an Alpine Go stage, and the final image is `scratch` with CA certificates copied from the build stage, running as non-root. The server speaks plain HTTP by default for use behind a proxy, and can serve a static certificate or provision Let's Encrypt certificates itself with certmagic.

## Considered Options

- Caddy serving static files plus a backend container (v1): rejected, extra overhead and orchestration for a single club.
- `x/crypto/acme/autocert`: rejected in favour of certmagic for renewal handling and future DNS-01 support.

## Consequences

The frontend can later be served from a CDN because the API is the contract and the SPA is plain static output. Non-root ACME needs host ports 80 and 443 mapped to the container's high ports.
