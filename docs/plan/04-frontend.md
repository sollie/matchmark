# MatchMark — Frontend Specification

## Technology Stack

| Concern | Choice |
|---------|--------|
| Framework | SvelteKit (adapter-static for PWA output) |
| PWA | `@vite-pwa/sveltekit` (Workbox-backed service worker) |
| Language | TypeScript |
| Styling | CSS custom properties; no CSS framework (outdoor high-contrast theme) |
| Offline storage | IndexedDB via `idb` helper |
| Build output | Static files → Docker volume → served by Caddy |

## Route Map

| Route | Access | Purpose |
|-------|--------|---------|
| `/login` | public | Username + password login |
| `/live` | public | List active competitions with spectator mode enabled; select one |
| `/live/[id]` | public | SSE-driven live ranked scoreboard for selected competition |
| `/` | auth | Redirect: admin → `/admin`, user → `/score/select` |
| `/score/select` | user | Pick active competition + own lane number |
| `/score/stage` | user | Scoring keypad — one stage at a time |
| `/score/summary` | user | Read-only summary when all stages locked |
| `/admin` | admin | Dashboard |
| `/admin/users` | admin | User CRUD |
| `/admin/competitions` | admin | Competition list |
| `/admin/competitions/new` | admin | Create competition + add shooters |
| `/admin/competitions/[id]` | admin | Detail: shooter list + live SSE monitor |
| `/admin/competitions/[id]/export` | admin | Export buttons (PDF / CSV / JSON) |

## Auth Flow

1. `POST /api/auth/login` → receive `access_token` + `refresh_token`
2. Store `access_token` in memory (Svelte store — never in localStorage)
3. Store `refresh_token` in `localStorage` (survives page reload)
4. `client.ts` attaches `Authorization: Bearer <access_token>` to every request
5. On 401: call `POST /api/auth/refresh` silently, retry original request
6. On refresh failure: clear stores, redirect to `/login`
7. Proactive refresh: set `setTimeout` to refresh 5 minutes before expiry

## Svelte Stores

### `stores/auth.ts`
```typescript
{
  accessToken: string | null,
  user: { id: number, username: string, role: 'user' | 'admin' } | null,
  expiresAt: number | null   // unix ms
}
```

### `stores/session.ts`
```typescript
{
  competitionId: number | null,
  competitionName: string | null,
  ownLane: number | null,
  scoreLane: number | null,   // derived: ownLane-1, wrapping
  totalStages: number | null
}
```

### `stores/scoring.ts`
```typescript
{
  currentStage: number,
  stages: StageState[],      // indexed by stage_number-1
}

type StageState = {
  stageNumber: number,
  expectedShots: number,
  distance: string,
  timeSec: number,
  position: string,
  hits: string[],            // ["X","10","9","8","7","0","M"]
  lockedByScorer: boolean,
  fullyLocked: boolean,
  stageScore: number,
  xCount: number,
  synced: boolean            // true if persisted to backend
}
```

## Offline Sync

### Queue Design (`lib/sync/queue.ts`)

```
IndexedDB store: "sync_queue"
  { id (autoincrement), competitionId, lane, stage, hits: string[], retries, timestamp }

Flow:
  1. Scorer enters/edits hits → written to IndexedDB + attempted POST immediately
  2. POST succeeds      → delete from IndexedDB; mark stage.synced=true in store
  3. POST fails (offline) → item stays in queue; Background Sync tag registered
  4. Service worker "sync" event → replay all queued items in order
  5. After 3 failed retries  → mark item as errored; show warning banner in UI
  6. iOS fallback (no Background Sync API):
     → listen to `visibilitychange`; on foreground, replay queue manually
```

### Service Worker (`service-worker.ts`)

```typescript
// Precache: all static assets (JS, CSS, HTML, icons, webmanifest)
// Runtime cache: NetworkFirst for /api/competitions (offline fallback)
// Sync: handle 'sync' event → replay IndexedDB queue
// Install: skipWaiting(); Activate: clients.claim()
```

## Scoring Stage UI

### Layout

```
┌─────────────────────────────────────────────────────┐
│  ● ● ● ◉ ○ ○ ○ ○ ○ ○      (stage progress dots)    │
│  Stage 4 of 10  ·  50 yd  ·  165s                   │
│  Scoring: Alice Hansen (lane 3)                      │
├─────────────────────────────────────────────────────┤
│              Shots: 11 / 24                          │
│              Score: 93    X: 2                       │
├────────┬────────┬────────┬────────┬────────┬─────────┤
│   X    │   10   │    9   │    8   │    7   │    0    │
│  [72px tall, high-contrast tap targets]              │
├────────┴────────┴────────┴────────┴────────┴─────────┤
│                        M                             │
│  [full-width red button, 72px tall]                  │
├──────────────┬──────────────────┬────────────────────┤
│  ← Undo      │  ← Prev Stage    │   Next Stage →     │
├──────────────┴──────────────────┴────────────────────┤
│                  [ Lock Stage ]                       │
│  (enabled only when shots == expected_shots)         │
└─────────────────────────────────────────────────────┘
```

### Behaviour Rules
- Only the current stage is interactive; previous stages are read-only (greyed keypad)
- `Undo` removes the last entered hit from the local array
- `Next Stage` disabled if current stage not locked and has unsaved hits
- `Next Stage` also disabled while the post-lock review modal is open and pending
- `Lock Stage` button disabled until `hits.length === expectedShots`
- After locking, stage becomes read-only; a post-lock review modal appears (see below)
- `Previous Stage` always enabled; shows previous stage in read-only mode

### Post-Lock Stage Review Modal

After a scorer locks a stage, a full-screen overlay appears showing their **own lane's score**
(i.e. the score recorded by the adjacent scorer). The modal stays open until the scorer taps
"Next →" — which is blocked until the adjacent scorer has also locked that stage.

**Flow:**
1. Scorer taps "Lock Stage" → stage is locked on the server → modal opens
2. Frontend calls `openSSEStream()` (fetch + ReadableStream with `Authorization: Bearer`) to
   `GET /api/competitions/{id}/lanes/{ownLane}/stage/{stageNum}/stream`
3. SSE stream emits updates every 3s; once `fully_locked === true` the stream closes automatically
4. Modal shows a spinner ("Waiting for your scorer to finish…") until `fully_locked` becomes true
5. Once resolved: displays score, X count, and hit sequence; "To protest this score, contact the match director."
6. Scorer taps "Next →" → modal resets; if all stages are fully locked → navigate to `/score/summary`

**Error state:** if the SSE connection fails, modal shows an error message
("Could not retrieve your score. Contact the match director.") and "Next →" is re-enabled.

### Colour / Accessibility
- Dark background (`#1a1a2e`) with high-contrast white/yellow text
- Minimum tap target: 72 × 72 px (exceeds WCAG 2.5.5 AAA 44px)
- `M` (miss) button: red background to emphasise weight of the entry
- Active stage dot highlighted; locked dots shown with checkmark

## PWA Configuration (`vite.config.ts`)

```typescript
VitePWA({
  registerType: 'autoUpdate',
  strategies: 'injectManifest',  // custom service-worker.ts
  srcDir: 'src',
  filename: 'service-worker.ts',
  workbox: {
    globPatterns: ['**/*.{js,css,html,ico,png,svg,webmanifest}'],
  },
  manifest: {
    name: 'MatchMark',
    short_name: 'MatchMark',
    description: 'WA1500/PPC Competition Scoring',
    theme_color: '#1a1a2e',
    background_color: '#1a1a2e',
    display: 'standalone',
    orientation: 'portrait-primary',
    start_url: '/',
    icons: [
      { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
      { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
      { src: '/icons/icon-512-maskable.png', sizes: '512x512',
        type: 'image/png', purpose: 'maskable' }
    ]
  }
})
```

## Admin Competition Monitor

- SSE connection to `GET /api/admin/competitions/{id}/monitor` (authenticated via `Authorization: Bearer` header)
- Uses `openSSEStream()` (`fetch` + `ReadableStream`) — no `EventSource`
- Auto-reconnects on disconnect (exponential backoff, max 30s)
- Live table columns: Lane | Shooter | Discipline | Stage | Score | X | Locked

### Monitor Table Actions (per row):
- Click row → slide-out panel with:
  - Stage-by-stage score breakdown
  - Per-stage: `[Edit Hits]` button (opens hit editor modal)
  - Per-stage: `[Unlock Stage]` button
  - `[Unlock All Stages]` at panel footer

### Hit Editor Modal (admin):
- Same keypad UI as scorer view but without lock enforcement
- Admin can modify any stage regardless of lock status
- Submits to `POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/hits`

## Spectator Live View

### `/live` — Competition Selector

- Fetches `GET /api/spectator/competitions` (no auth required)
- Displays a card per competition: name and date
- Clicking a card navigates to `/live/[id]`
- No auth guard; accessible without an account

### `/live/[id]` — Live Scoreboard

- Opens SSE connection to `GET /api/spectator/competitions/{id}/live` (no auth)
- Auto-reconnects on disconnect (exponential backoff, max 30s — same pattern as admin monitor)
- Shows "Competition not found or not available for public viewing" if SSE returns 404
- Table sorted by cumulative score descending (live ranking); rank updates in real time

#### Scoreboard columns:
Rank | Lane | Name | Club | Discipline | Weapon Class | Stage 1 … Stage N | Total | X

- Stage columns are dynamic based on `total_stages` from first SSE event
- Stage cells fill in as scores arrive; stages not yet completed shown as grey/empty
- Both routes excluded from auth guard in `+layout.svelte`
