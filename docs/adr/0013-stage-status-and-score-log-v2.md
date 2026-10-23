---
status: accepted
---

# Stage status replaces the locked flag; Score log v2

A lock used to be final. With shooter acceptance (see `GLOSSARY.md`: Awaiting acceptance, Accepted, Dispute) the scorer's lock only proposes a result, so a stage needs a status rather than a boolean.

`stages.status` is `open`, `awaiting_acceptance` or `accepted` (migration 016, which maps `locked = 1` to `accepted` and drops `locked`). `accepted_by`, `accepted_at` and `disputed_at` sit beside it. Transitions are conditional updates inside one transaction that also inserts a row into `stage_events`, so a lost race returns a conflict and the event history never disagrees with the stage:

- device lock: `open` to `awaiting_acceptance`
- admin lock as the signed-in user: `open` to `accepted`
- admin accept: `awaiting_acceptance` to `accepted`
- device accept (the shooter's own lane only): `awaiting_acceptance` to `accepted`, `accepted_by` is the lane
- device dispute (the shooter's own lane only): `awaiting_acceptance` to `open`, hits kept, `disputed_at` set and cleared by the next lock
- admin unlock: any non-open status to `open`, hits kept
- admin overwrite: an open stage only changes counts; an awaiting or accepted stage ends accepted, taken over by the user, with an event only when it was awaiting

Only accepted stages count. Standings, progress, the `.xlsx` report and the CSV treat other stages as zero; the admin entries list still shows every stage's score. The device phase (`scoring`, `waiting`, `needs_acceptance`) is derived server-side from the stages of the score lane and of the device's own lane. Lockstep and the current-stage pointer are separate steps of the shooter-acceptance work.

## Score log v2

The Score log has one event, `stage_status`, with `schema_version` 2 and a `status` field, replacing `stage_locked` and `stage_unlocked`. An unlock is a line with status `open`, its ring counts and no lock provenance, so replay no longer deletes stages: the latest line per stage always wins, and a restored open stage keeps its hits. New fields are `accepted_by`, `accepted_at` and `disputed_at`. `restore` accepts only version 2 and fails on version 1 lines, since a log of mixed versions has no consistent replay; older logs are restored with the earlier release. A restored competition is `finished` when every stage is accepted and `active` otherwise.

## Considered Options

- Keeping `stage_locked` and adding new events: rejected, one event with a status keeps the replay rule a single last-line-wins fold.
- Reading old lines as accepted: rejected, version 1 carried no acceptance semantics and mixing versions would hide that.

## Consequences

The lock endpoints return `{"status": ...}` instead of `{"locked": true}`, and the spectator `locked` fields are now `accepted`. The CSV column `locked` is `accepted`. The audit log records `stage.lock`, `stage.accept`, `stage.dispute` and `stage.unlock`.
