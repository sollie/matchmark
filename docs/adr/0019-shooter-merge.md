---
status: accepted
---

# Merging duplicate shooters

A shooter with any entry can never be deleted, so a duplicate created by mistake would stay in the registry for good. We add an admin-only merge that absorbs a duplicate (source) into the real shooter (target) in one transaction: every entry of the source is re-pointed to the target, then the source row is deleted.

The target wins on every field. The source's WAID carries over only when the target has none. Entries in finished matches are moved too, and the shooter name stored on stage and score records is refreshed; reports already sent are not regenerated.

The merge is refused when both shooters have different WAIDs (two real people) or when both have an entry in the same match, because `UNIQUE (match_id, shooter_id)` would otherwise force us to drop scores silently. The conflicting matches are listed so the admin can withdraw or fix them first.

The source row is deleted and cannot be restored, and entries in finished matches change owner, so a merge is irreversible. The audit log (`shooter.merge`, with source and target) is the only record. Archiving, which is a reversible flag, covers shooters who have left the club and is not an ADR.
