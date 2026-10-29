---
status: superseded by 0021 (counts are per scoring segment)
---

# Hits are stored as ring counts per stage

The official scorecard records how many hits fell in each ring (X, 10, 9, 8, 7, 0, miss) per match, not shot order, and the tie-break rules only need counts. We store seven counts per stage that must sum to the stage's expected shots, instead of one row per shot as in v1. This simplifies the schema and the scorer screen and mirrors the paper card.

## Consequences

Shot order and per-shot corrections are not available. If shot-level data is ever needed, it requires a new table and a migration.
