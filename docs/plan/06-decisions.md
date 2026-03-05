# MatchMark — Technology & Design Decisions

## Technology Choices

| Concern | Decision | Rationale |
|---------|----------|-----------|
| Frontend framework | SvelteKit + `@vite-pwa/sveltekit` | Compiled output, minimal runtime, excellent PWA plugin support, adapter-static produces a clean bundle for Caddy to serve |
| Backend language | Go 1.26+ | Strong concurrency, small binaries, no runtime dependency, fast cold start |
| HTTP router | `github.com/go-chi/chi/v5` | Lightweight, idiomatic Go, clean middleware composition, no magic |
| SQLite driver | `modernc.org/sqlite` | Pure Go (no CGO), implements `database/sql`, simple Docker builds with `CGO_ENABLED=0` |
| Password hashing | argon2id | OWASP-recommended; memory-hard; resistant to GPU/ASIC brute force |
| JWT algorithm | HS256 | Sufficient for a single-backend deployment; no asymmetric key management needed |
| Access token TTL | 1 hour | Covers a typical slow competition session; short enough to limit exposure |
| Refresh token TTL | 7 days | Allows multi-day competition events without re-login; rotated on every use |
| Refresh token storage | DB (hashed) | Enables server-side revocation (logout, admin force-logout) |
| Refresh token families | `family_id` column on `refresh_tokens` | Detects token replay: if an already-consumed token from a family is presented, the entire family is revoked |
| PDF export | `github.com/go-pdf/fpdf` | Pure Go, no CGO, mature, sufficient layout control for scorecard format |
| Admin monitoring | SSE (Server-Sent Events) | Simpler than WebSocket for a unidirectional push use-case |
| SSE authentication | `fetch` + `ReadableStream` with `Authorization: Bearer` | Unlike `EventSource`, `fetch`-based streaming supports custom request headers, eliminating the need for a short-lived ticket system; Safari 16.4+ supports response body streaming |
| SSE poll interval | 3 seconds | Frequent enough for live feel; low enough not to strain SQLite under concurrent reads |
| Offline storage | IndexedDB + Background Sync API | Survives app kills, page reloads, connection drops; Background Sync defers replay to service worker |
| iOS offline fallback | `visibilitychange` replay | Background Sync not available on iOS; foreground event covers the common case |
| Docker service count | 2 services (backend + caddy) | The `caddy/Dockerfile` is a multi-stage build: Node builds the SvelteKit app in stage 1, caddy copies the output in stage 2. Eliminates the need for a separate frontend service and a shared `frontend_dist` named volume, simplifying orchestration. |

## Domain Configuration

The Caddy domain is passed as the `DOMAIN` environment variable to the `caddy` container.
The Caddyfile uses Caddy's native `{$DOMAIN}` substitution syntax — no image rebuild
required when changing the domain for a new deployment.

```yaml
caddy:
  environment:
    - DOMAIN=${DOMAIN}
```

```caddyfile
{$DOMAIN} {
    ...
}
```

## Design Decisions

### One scorer per lane
Each physical lane has exactly one user who scores the shooter in that lane. This
simplifies `stage_locks`: `fully_locked` flips to `1` the moment the single scorer
locks. Admins are exempt from lock checks and can overwrite any stage's hits directly.

### Score lane computed dynamically
`scoreLane = ownLane - 1` (lane 1 wraps to `lanes_total`) is computed at request time
from the JWT identity and the competition's `lanes_total`. It is never stored. This
prevents stale data if an admin changes lane assignments after sessions begin.

### User picks lane at session start
Users are not pre-assigned to lanes in the DB. They choose their own lane when
selecting a competition. The system does not prevent two users from claiming the same
lane (range etiquette governs this); one concurrent scorer per lane is the expected
physical constraint.

### Competition lifecycle: draft → active → finished
- `draft`: admin-only visibility; safe to add/edit shooters before the competition day
- `active`: visible to scorers; scoring enabled
- `finished`: read-only for all; export still available to admin
This prevents scorers from accidentally landing in a test or past competition.

### Hits stored as individual rows
Each shot is a separate `hits` row. Submitting a stage POSTs the full hit list which
replaces all existing hits in a transaction. This keeps the API simple (no patch/undo
endpoints needed server-side) and makes per-ring counts trivial SQL aggregations.

### Stages created eagerly on shooter add
When a shooter is added, all stage rows are immediately created with `expected_shots`
derived from their discipline and weapon class. The scoring API always finds an
existing stage row — no lazy creation, no ambiguity.

### 5-shot revolver variant auto-detected
`expected_shots` for 48-discipline stages is set at shooter-creation time based on
whether `weapon_class == "Standard Revolver 2.75\" 5-Shot"`. No separate `40`
discipline needed; the variant is transparent to the scoring UI.

### All strings in English
All UI labels, PDF export labels, API field names, and code strings are in English.
The rulebook data (weapon class names, discipline identifiers) uses the WA1500 official
English naming from the 2026-01-01 rulebook edition.

### Refresh token rotation
Every token refresh issues a new refresh token and deletes the old one. This limits the
window of token theft and ensures that a logged-out session cannot be reactivated.

### Admin password minimum 16 characters
Enforced at seed time and on every password set/reset via the service layer. Future
work: 2FA / passkeys (not in scope v1).

### Single-lane competitions require admin scoring
`GetScoreLane(1, 1)` returns `1` — a one-lane competition causes the scorer to score
themselves (lane 1 wraps to lane 1). To prevent accidental self-scoring, `SubmitHits`
and `LockStage` return `ErrSingleLaneAdminOnly` when `comp.LanesTotal == 1` and the
caller is not an admin. The frontend surfaces this as a non-dismissible red banner
with blocked keypad interaction. Read access (`GetStageDetail`) is not restricted.
