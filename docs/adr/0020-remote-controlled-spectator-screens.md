---
status: accepted
---

# Remote-controlled spectator screens

Spectator displays far from the admins could not be switched between matches without walking to them. A spectator display can now opt in to remote control and become a Screen; an admin assigns it a Match, or the same Match to all Screens. This extends ADR 0001: besides paired scorer devices there is now a second kind of unauthenticated-by-account client that holds a token, but a Screen can only receive an assignment, never write results.

A Screen registers itself, receives a random 128-bit token (stored hashed server-side, kept in the browser's `localStorage`) and a short code shown on its display. It appears in the admin list only after an admin approves the code and names it. Registration is capped and rate limited per client IP, reusing the login limiter and `TRUSTED_PROXY` handling. A Screen learns its assignment over a token-authenticated SSE stream (ADR 0010) that carries only which Match to show; the Screen then subscribes to the existing spectator stream, so visibility flags apply unchanged. A draft Match shows as a Waiting screen with no identifying data and is pushed to the Screen when the Match starts. Screens unseen for 24 hours are purged, and a viewer can leave remote control, which removes the registration.

## Considered Options

- Open self-registration: rejected, anyone reaching the server could fill and spoof the admin's list.
- Pairing link or QR like scorer devices: rejected, a wall display has no camera or keyboard to open a link and admins would have to visit it, which is the problem being solved.
- Polling or WebSocket for assignments: rejected for the reasons in ADR 0010.

## Consequences

The spectator surface gains a small write path (register, heartbeat, leave) and a background purge. Assignments are explicit per-Screen rows; "apply to all" writes them once and does not persist as a setting.
