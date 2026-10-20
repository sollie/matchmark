---
status: accepted
---

# Live updates over server-sent events

Spectator and admin screens polled every 3-5 seconds, which costs too much cellular data over a 12-hour range day. The server now pushes live updates over SSE (`text/event-stream`) on one stream per role and competition: spectator, admin and scorer. This supersedes "SSE out for v1" in the spec.

Every stream starts with a full snapshot, so reconnecting needs no replay and no `Last-Event-ID`. Spectator streams push the standings and progress snapshot. Admin streams push a `changed` ping and the SPA refetches entries. Scorer streams push the scorer session and end with a `revoked` event when the pairing is revoked or the competition finishes. The server sends a heartbeat comment every 25 seconds and closes streams after 30 minutes; the SPA reconnects with backoff, closes the stream while the tab is hidden, and does not fall back to polling.

Scorers authenticate with a device token header, which `EventSource` cannot send, so one small `fetch`-based SSE client serves all three roles. An in-memory broadcaster in `service` publishes after each successful write; spectator snapshots are built once per change and fanned out.

## Considered Options

WebSocket was rejected: all traffic is server to client, it needs its own compression handling and a dependency, and SSE gives cookie auth and the existing compression middleware for free.
