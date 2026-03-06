# MatchMark — Backend Specification

## Technology Stack

| Concern | Choice |
|---------|--------|
| Language | Go 1.26+ |
| HTTP router | `github.com/go-chi/chi/v5` |
| SQLite driver | `modernc.org/sqlite` (pure Go, `database/sql` compatible) |
| Password hashing | `golang.org/x/crypto` argon2id |
| JWT | `github.com/golang-jwt/jwt/v5` |
| PDF export | `github.com/go-pdf/fpdf` |
| Linting | `golangci-lint` |
| Vulnerability scan | `trivy --scanners vuln,secret` |

## Layer Responsibilities

```
handler/    parse HTTP request → call service → marshal JSON response
service/    business rules: validation, lane derivation, lock logic, rulebook lookups
repository/ SQL queries via database/sql; no business logic; transactions exposed
model/      shared Go structs (Competition, Shooter, Stage, Hit, StageLock, User)
auth/       JWT sign/verify/refresh, argon2id hash/verify, chi middleware
export/     PDF (gofpdf), CSV (encoding/csv), JSON (encoding/json)
rulebook/   static course-of-fire definitions, discipline/weapon validation
db/         migration runner (embedded SQL), admin seed
```

## Complete API Contract

### Auth

```
POST /api/auth/login
     Body:     { "username": string, "password": string }
     Response: { "access_token": string, "expires_in": 3600,
                 "user": { "id", "username", "role" } }
     Sets:     HttpOnly; Secure; SameSite=Lax cookie `mm_rt` containing the refresh token

POST /api/auth/refresh
     Cookie:   mm_rt (HttpOnly refresh token)
     Response: { "access_token": string, "expires_in": 3600 }
     Sets:     Rotated mm_rt cookie

POST /api/auth/logout
     Headers:  Authorization: Bearer <token>
     Cookie:   mm_rt (read to revoke token; clears cookie server-side)
     Response: 204 No Content

Access tokens include a user auth version claim. Every authenticated request reloads the
current user and rejects tokens for deleted users, changed roles, or bumped auth versions.
Password resets and admin role changes revoke all refresh tokens for that user and bump the
auth version so existing access tokens stop working immediately.
```

### Current User

```
GET /api/users/me
    Headers:  Authorization: Bearer <token>
    Response: { "id": int, "username": string, "role": string }
```

### User Management (admin only)

```
GET    /api/users
       Response: [ { "id", "username", "role", "created_at" } ]

POST   /api/users
       Body:     { "username": string, "password": string, "role": "user"|"admin" }
       Response: 201 { "id", "username", "role" }

PUT    /api/users/{id}
       Body:     { "username"?: string, "role"?: string, "password"?: string }
       Response: 200 { "id", "username", "role" }

DELETE /api/users/{id}
       Response: 204

POST   /api/users/{id}/reset-password
       Body:     { "new_password": string }   -- admin sets directly; min 16 chars
       Response: 204
```

### Competitions

```
GET  /api/competitions
     Query:    ?status=draft|active|finished  (admins only; scorers are forced to active)
     Response: [ { "id", "name", "date", "lanes_total", "status", "spectator_enabled" } ]

POST /api/competitions                          (admin)
     Body:     { "name": string, "date": "YYYY-MM-DD", "lanes_total": int,
                 "spectator_enabled"?: bool }
     Response: 201 { "id", "name", "date", "lanes_total", "status": "draft",
                     "spectator_enabled" }

GET  /api/competitions/{id}
     Auth:     admins may read any competition; scorers may read active competitions only
     Response: { "id", "name", "date", "lanes_total", "status", "spectator_enabled" }

PUT  /api/competitions/{id}                     (admin)
     Body:     { "name"?, "date"?, "lanes_total"?, "status"?, "spectator_enabled"? }
     Response: 200 { "id", "name", "date", "lanes_total", "status", "spectator_enabled" }
```

### Shooters / Lanes

```
GET  /api/competitions/{id}/shooters
     Auth:     admin → full shooter payload for any competition
               scorer → active competitions only, reduced payload
     Response: admin  [ { "id", "name", "wa_id", "club", "lane_number",
                          "discipline", "weapon_class" } ]
               scorer [ { "id", "name", "lane_number", "discipline" } ]

POST /api/competitions/{id}/shooters            (admin)
     Body:     { "name": string, "wa_id": string, "club"?: string,
                 "lane_number": int, "discipline": "1500"|"60"|"48",
                 "weapon_class": string }
     Validation:
       - weapon_class must be valid for discipline (rulebook.ValidateWeaponClass)
       - lane_number must be unique within competition
       - lane_number must be 1..lanes_total
     Side-effect: creates all stage rows for this shooter (expected_shots from rulebook)
     Response: 201 { "id", "name", "wa_id", "club", "lane_number",
                     "discipline", "weapon_class" }

PUT  /api/competitions/{id}/shooters/{sid}      (admin)
     Body:     { "name"?, "wa_id"?, "club"?, "lane_number"?,
                 "discipline"?, "weapon_class"? }
     Response: 200 { ... }

DELETE /api/competitions/{id}/shooters/{sid}    (admin)
     Response: 204 (cascades: stages → hits, stage_locks)
```

### Scorer Assignments

```
POST /api/competitions/{id}/assign
     Auth: authenticated user (scorer)
     Body: { "shooter_id": int }
     Effect: claims shooter for this user within the competition.
             Fails with 409 if the shooter is already claimed by another user,
             or if this user already has an assignment in the competition.
     Response: 200 {
       "id", "competition_id", "shooter_id", "user_id",
       "username", "shooter_name", "lane_number", "assigned_at"
     }

GET /api/competitions/{id}/assign
    Auth: authenticated user
    Effect: returns the caller's current assignment, for auto-restore on re-login.
    Response: 200 { same shape as POST response }
              404 if no assignment found

GET /api/competitions/{id}/assignments
    Auth: authenticated user
    Response: [ { "id", "competition_id", "shooter_id", "user_id",
                  "username", "shooter_name", "lane_number", "assigned_at" } ]
    Purpose: allows UI and admin monitor to show which scorer is locked to which lane.

DELETE /api/competitions/{id}/assign
       Auth: admin only
       Body: { "shooter_id": int }
       Effect: releases the assignment for the given shooter so another user can claim it.
       Response: 204
```

### Scoring

```
GET /api/competitions/{id}/lanes/{lane}/stage/{stage}
    Auth: authenticated user (any role may read any lane)
    Response: {
      "stage_number":    int,
      "expected_shots":  int,
      "distance":        string,
      "time_seconds":    int,
      "position":        string,
      "discipline":      string,
      "total_stages":    int,
      "shooter": { "name", "wa_id", "club" },
      "hits":            [ { "ring": string, "points": int } ],
      "locked_by_scorer": bool,
      "fully_locked":    bool,
      "stage_score":     int,
      "stage_x_count":   int,
      "cumulative_score": int,
      "cumulative_x_count": int
    }

POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/hits
     Auth: scorer assigned to that lane OR admin
     Body: { "hits": [ "X"|"10"|"9"|"8"|"7"|"0"|"M", ... ] }
     Rules (scorer): must have a scorer_assignment for this competition whose
                     lane_number matches the requested lane;
                     stage must not be locked; len(hits) must == expected_shots;
                     if competition has only one lane → 403 "admin scoring required"
     Rules (admin):  no assignment or lock check; len(hits) must == expected_shots
     Effect: deletes all existing hits for this stage, inserts new ones (transaction)
     Response: 200 {
       "hits":        [ { "ring", "points" } ],
       "stage_score": int,
       "x_count":     int
     }

POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/lock
     Auth: scorer assigned to that lane
     Rules (scorer): if competition has only one lane → 403 "admin scoring required"
     Effect: sets stage_locks.locked=1 for (stage_id, user_id);
             sets stages.fully_locked=1 (one scorer per lane invariant)
     Response: 200 { "fully_locked": true }

POST /api/competitions/{id}/lanes/{lane}/stage/{stage}/unlock
     Auth: admin only
     Effect: sets stage_locks.locked=0; sets stages.fully_locked=0
     Response: 200 { "fully_locked": false }
```

### Scorer Stage Review (SSE)

```
GET /api/competitions/{id}/lanes/{lane}/stage/{stage}/stream
    Auth: Authorization: Bearer <access_token>
    Content-Type: text/event-stream
    On connect: verifies lane assignment (same check as SubmitHits); 403 if not assigned.
    Emits every 3 seconds until fully_locked == true, then closes:
    data: {
      "stage_score":   int,
      "stage_x_count": int,
      "hits":          [ { "ring": string, "points": int } ],
      "fully_locked":  bool
    }
    401 if token missing or invalid.
    403 if caller is not assigned to the requested lane.
    Stream closes automatically once fully_locked becomes true (final frame emitted).
```

### Admin Monitoring (SSE)

```
GET /api/admin/competitions/{id}/monitor
    Auth: Authorization: Bearer <access_token>  (admin role required)
    Content-Type: text/event-stream
    Emits every 3 seconds:
    data: {
      "competition_id":   int,
      "competition_name": string,
      "status":           string,
      "lanes": [
        {
          "lane_number":        int,
          "shooter_name":       string,
          "discipline":         string,
          "weapon_class":       string,
          "current_stage":      int,   -- first unlocked stage; total_stages+1 if all done
          "stage_fully_locked": bool,
          "scorer_locked":      bool,
          "cumulative_score":   int,
          "x_count":            int
        }
      ]
    }
```

### Spectator (public — no auth required)

```
GET /api/spectator/competitions
    Response: [ { "id", "name", "date", "status" } ]
    Returns only competitions where spectator_enabled = 1 AND status = 'active'

GET /api/spectator/competitions/{id}/live
    Content-Type: text/event-stream
    Auth: none
    Returns 404 if competition not found or spectator_enabled = 0
    Emits every 3 seconds:
    data: {
      "competition_id":   int,
      "competition_name": string,
      "lanes": [
        {
          "lane_number":        int,
          "shooter_name":       string,
          "club":               string | null,
          "discipline":         string,
          "weapon_class":       string,
          "cumulative_score":   int,
          "cumulative_x_count": int,
          "stages": [
            { "stage_number": int, "score": int, "x_count": int, "locked": bool }
          ]
        }
      ]
    }
```

### Export (admin only)

```
GET /api/competitions/{id}/export/pdf
    Response: application/pdf; Content-Disposition: attachment

GET /api/competitions/{id}/export/csv
    Response: text/csv; Content-Disposition: attachment

GET /api/competitions/{id}/export/json
    Response: application/json
    Body: [
      {
        "name", "wa_id", "club", "discipline", "weapon_class",
        "total_score": int, "x_count": int,
        "per_ring":    { "X": int, "10": int, "9": int, "8": int,
                         "7": int, "0": int, "M": int },
        "stages": [
          {
            "stage_number": int, "score": int, "x_count": int,
            "per_ring":     { ... }
          }
        ]
      }
    ]
```

## Auth & Security Details

### JWT
- Algorithm: HS256
- Claims: `iss=matchmark`, `aud=matchmark-api`, `uid`, `sub` (username), `role`, `iat`, `exp`
- Access token TTL: 1 hour
- Refresh token TTL: 7 days; stored as `SHA-256(raw_token)` in `refresh_tokens` table
- Every refresh rotates the refresh token (old deleted, new inserted)
- `JWT_SECRET` env var must be ≥ 32 bytes; server refuses to start if missing or too short

### Argon2id Parameters
- `time=1, memory=65536 (64 MB), threads=4, keyLen=32`
- Meets OWASP minimum recommendations

### Password Policy
- Minimum 16 characters; enforced in service layer on create and reset
- Future: 2FA / passkeys (not in scope v1)

### Middleware Chain (chi)
```
Router
└── /api/*
    ├── RequireAuth (all /api routes except /api/auth/*)
    │   └── injects userID + role into request context
    └── RequireAdmin (admin-only routes)
        └── checks role == "admin", returns 403 otherwise
```

### Score Lane Authorization
- The `lane` path parameter in scoring endpoints refers to the **shooter's lane**.
- Any authenticated user may read (GET) any lane — no lane restriction on reads.
- To submit hits or lock a stage, a non-admin caller must have a row in
  `scorer_assignments` for the competition whose `lane_number` matches the requested lane.
  If not found or mismatched → 403.
- Admins bypass the assignment check entirely and may write to any lane.
- Lane assignment is done by the scorer selecting their name from the competition's
  shooter list; the server enforces uniqueness (one shooter per user, one user per shooter).

## PDF Export Layout

Mirrors the WA1500 official scorecard format, with all labels in English.

### Per-page structure (one page per shooter):
```
Header:   [Competition name]  [Date]
          Name: [name]   WA-ID: [wa_id]   Club: [club]
          Discipline: [1500|60|48]   Weapon Class: [weapon_class]

Score table:
          Stage | Points | X | 10 | 9 | 8 | 7 | 0 | M
          ------+--------+---+----+---+---+---+---+---
          (rows per stage)
          ------+--------+---+----+---+---+---+---+---
          Subtotals per match group (see groupings below)
          Grand total

Footer:   Scorer: ____________   Shooter: ____________   Match Director: ____________
```

### Stage groupings for subtotals:
- **48-shot**:  stages 1–2 (subtotal), stages 3–4 (subtotal), Total
- **60-shot**:  stages 1–2 (subtotal), stages 3–4 (subtotal), Total
- **1500**:     Match 1 (st.1–2), Match 2 (st.3), Match 3 (st.4),
                Match 4 (st.5–6), Match 5 (st.7–10), Total

## Startup Sequence

1. Load and validate env vars (`JWT_SECRET`, `ADMIN_PASSWORD`, `DB_PATH`)
2. Open SQLite DB (`modernc.org/sqlite` via `database/sql`)
3. Run embedded migrations (idempotent — `CREATE TABLE IF NOT EXISTS`)
4. Seed admin user if no admin exists:
   - username: `admin`
   - password: from `ADMIN_PASSWORD` env var (min 16 chars enforced)
5. Start chi HTTP server on `:8080`
