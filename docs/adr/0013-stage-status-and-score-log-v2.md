---
status: accepted; stage status, lockstep and dispute history superseded by 0021 (per scoring segment)
---

# Stage status replaces the locked flag; Score log v2

A lock used to be final. With shooter acceptance (see `GLOSSARY.md`: Awaiting acceptance, Accepted, Dispute) the scorer's lock only proposes a result, so a stage needs a status rather than a boolean.

`stages.status` is `open`, `awaiting_acceptance` or `accepted` (migration 016 before the squash). `accepted_by`, `accepted_at` and `disputed_at` sit beside it. Transitions are conditional updates inside one transaction that also inserts a row into `stage_events`, so a lost race returns a conflict and the event history never disagrees with the stage:

- device lock: `open` to `awaiting_acceptance`
- admin lock as the signed-in user: `open` to `accepted`
- admin accept: `awaiting_acceptance` to `accepted`
- device accept (the shooter's own lane only): `awaiting_acceptance` to `accepted`, `accepted_by` is the lane
- device dispute (the shooter's own lane only): `awaiting_acceptance` to `open`, hits kept, `disputed_at` set and cleared by the next lock
- admin unlock: any non-open status to `open`, hits kept
- admin overwrite: an open stage only changes counts; an awaiting or accepted stage ends accepted, taken over by the user, with an event only when it was awaiting

Only accepted stages count. Standings, progress, the `.xlsx` report and the CSV treat other stages as zero; the admin entries list still shows every stage's score. The device phase (`scoring`, `waiting`, `needs_acceptance`, `next_stage`, `finished`) is derived server-side from the current stage of the score lane and of the device's own lane.

## Lockstep

The match stores a current-stage pointer (`current_stage`, 1 at activation). Only the admin advances it, and only when every manned, non-withdrawn lane has accepted the current stage (the gate). Devices act only on the current stage; admin corrections of earlier stages do not move the pointer. Accepting the last stage finishes the match through the existing finish path, which revokes pairings; devices still receive a minimal `finished` session so they can show a Finished screen. Status changes are conditional updates (`from` status), so a concurrent double finish is a no-op. Entries are added and moved to another lane only in draft (an active match still allows a category change); withdrawal is allowed only at the gate. A restore sets the pointer to the lowest stage with a non-accepted entry stage, or the last stage when everything is accepted.

## Control room

The admin sees the current stage as one row per manned lane from `GET /api/matches/{id}/control-room`, and `POST /api/matches/{id}/accept-all` accepts every awaiting stage when no lane is open. The best-effort connected state is kept in memory by counting open scorer streams per lane, because the broadcaster already knows who is subscribed. It is never stored, so a sleeping phone or a restart shows as not connected.

## Dispute history

`GET /api/matches/{id}/stage-events` serves the `stage_events` table to admins, filtered by exact scorer, shooter and stage, with lock and dispute counts per scorer computed over the filtered events. Device events already store the shooter and scorer resolved from the pairing at the time, so the history stays correct after pairings are revoked.

## Score log v2

The Score log has one event, `stage_status`, with `schema_version` 2 (version 3 since ADR 0017, which adds a `competition` object beside `match`) and a `status` field, replacing `stage_locked` and `stage_unlocked`. An unlock is a line with status `open`, its ring counts and no lock provenance, so replay no longer deletes stages: the latest line per stage always wins, and a restored open stage keeps its hits. New fields are `accepted_by`, `accepted_at` and `disputed_at`. `restore` accepts only version 2 and fails on version 1 lines, since a log of mixed versions has no consistent replay; older logs are restored with the earlier release. A restored match is `finished` when every stage is accepted and `active` otherwise.

## Considered Options

- Keeping `stage_locked` and adding new events: rejected, one event with a status keeps the replay rule a single last-line-wins fold.
- Reading old lines as accepted: rejected, version 1 carried no acceptance semantics and mixing versions would hide that.

## Consequences

The lock endpoints return `{"status": ...}` instead of `{"locked": true}`, and the spectator `locked` fields are now `accepted`. The CSV column `locked` is `accepted`. The audit log records `stage.lock`, `stage.accept`, `stage.dispute` and `stage.unlock`.
