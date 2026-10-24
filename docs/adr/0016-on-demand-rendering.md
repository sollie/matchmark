---
status: accepted
---

# Render scorecards and results on demand, never store them

Scorecard copies, the Results PDF and later renderings are produced from the current database rows for every request and are never written to disk or the database. They always reflect the current data, backups and replication stay small, and there is no invalidation to get wrong.

- **PNG:** `golang.org/x/image` draws the Scorecard copy (`font`, `opentype` and `draw`) with the Go fonts (`golang.org/x/image/font/gofont`, BSD-style licence from the Go project) embedded in the binary. Both are pure Go, so `CGO_ENABLED=0` and the Alpine image keep working. Row and shot counts come only from `internal/rulebook`.
- **PDF:** `github.com/johnfercher/maroto/v2` lays out the Results PDF (A4 portrait, one page per category, repeating header and footer). It is pure Go, so `CGO_ENABLED=0` holds, and it is actively maintained, unlike the archived `fpdf` and `gofpdf`. The same Go fonts are registered as UTF-8 fonts so names with non-ASCII letters render. The public and admin endpoints share one renderer (`report.ResultsPDF`) fed by `service.PublicResults` or `service.AdminResults`; only the public one goes through the render guard, because the admin one is authenticated and audited per download, which request coalescing would merge.
- **Render protection:** rendering costs CPU and memory and the endpoints are public, so they sit behind one reusable middleware, `handler.RenderGuard` (`internal/handler/renderguard.go`). It combines a global semaphore (4 concurrent renders, a queue of 8 waiting up to 2 seconds, then `503` with `Retry-After`), coalescing of identical requests with `golang.org/x/sync/singleflight` (the same request URI shares one render and one response), and a fixed-window limit of 20 requests per minute per client IP and resource (`429` with `Retry-After`). The limit is never keyed on the IP alone because a whole shooting range shares one wifi address. The client IP is the one the server already resolves with `TRUSTED_PROXY` and `X-Forwarded-For`.

## Consequences

Each request recomputes the image, which the guard and the no-cache headers accept. The rate limit state is in memory and per process (the deployment is one binary); edge rate limiting stays with the CDN. `RenderGuard.Middleware(resource)` takes a function that names the resource, so the Results PDF reuses the same guard with its own keys; endpoints that share one guard share its global semaphore.
