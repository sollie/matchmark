---
status: accepted
---

# Scoring segments

The PPC cheat sheet (`docs/references/ppc1500-sverige-lathund-gb.pdf`) and the score cards (`docs/references/Skjutkort-*.pdf`) score at fixed "Score the target" points, each covering one or more stages in one go. MatchMark scored, locked, accepted and gated every stage on its own, which is finer than the paper and shows stage distances the scorer never needs. We follow the paper and use its names.

A **scoring segment** is a run of stages between two "Score the target" lines. One set of seven ring counts is recorded per segment and must sum to the segment's shots; nothing is recorded per stage. Lock, accept, dispute, the gate, the control room, the current segment, the Dispute history and the Score log all work per segment. Segment names, stage lists and shot counts live only in `internal/rulebook` and are served by `GET /api/rulebook`.

- 150 course: Match 1 (24 shots), Match 2 (18), Match 3 (24), Match 4 (24), Match 5 · Stage 1+2 (30), Match 5 · Stage 3+4 (30).
- 48, 60 and 40 courses: Stage 1+2 and Stage 3+4, with 18 + 30, 30 + 30 and 15 + 25 shots. The score cards say "St. 1+2" and "St. 3+4" while the cheat sheet says Match 1 and Match 2; the score-card names win.

The UI shows the PDF names. The rulebook's "Match N" is the glossary Part, so our container for one course stays internally a Match and is shown by its name plus course label. Stage is now a descriptive rulebook term only.

Nothing is deployed, so migrations are squashed and the Score log `schema_version` is bumped without a compatibility layer. Withdrawal stays allowed between segments, and an admin edit works in any non-draft match.

## Supersedes

- ADR 0002 (ring counts per stage): counts are now per segment.
- The stage-status and lockstep parts of ADR 0013: status, lock, accept, dispute, the current pointer, the gate and the Dispute history move from stage to segment.
- The naming reasoning of ADR 0017: Part stays the rulebook's "Match N", but the UI now uses those names for scoring segments.
