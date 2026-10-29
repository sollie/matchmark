---
status: accepted; naming reasoning for Parts superseded by 0021
---

# A Competition contains Matches

The rulebook uses competition for the announced event of one or more days and match for one course fired within it (statement 1326), and limits what a shooter may fire per competition. MatchMark's "Competition" (one course, one date, one entry per shooter) is a rulebook match, and a club day with several courses was several unlinked Competitions. We rename it to Match and make Competition a container with a name and a date range; a Match belongs to exactly one Competition, and the rulebook's "Match 1" to "Match 5" stage groups become Parts.

Nothing is deployed, so the rename is done at once with squashed migrations and Score log `schema_version` 3 rather than through a compatibility layer. Entries, lanes, stages, visibility and reports stay per Match; Competition has no stored status; the per-competition limit of statement 1326 is not enforced because the club runs no re-entry matches.

## Considered Options

Keeping "Competition" for a single course and documenting the mapping was rejected: the next club day with two courses would need the container anyway. A Tournament container was rejected because the rulebook defines a tournament as a series of matches under one program, which is not what a club day is.
