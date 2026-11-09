---
status: accepted
---

# Entry snapshot at match start

Classification, class, club and country are frozen per entry when a match goes draft to active, so later reclassifications or club moves no longer change that match's results (supersedes the live-values consequence of ADR 0023).

- `entries` gains nullable `classification_at_match` (0 to 5), `class` (0 to 5), `club` and `country` (migration 006). Only the five 1500 categories get a classification and class. A non-NULL `club` marks the entry as frozen.
- Class equals the classification, except Unclassified, which maps to the club setting `unclassified_class` (`Master` default, or `Unclassified`).
- Back to draft clears the snapshot. A category change on a frozen entry retakes it. Matches finished before this change stay unfrozen and are not back-filled.
- Admins can correct `classification_at_match` and `class` of a frozen entry with `PUT /api/matches/{id}/entries/{entryID}/classification`; the change is audited as `entry.classification`.
- Standings, results, the CSV, the `.xlsx` and the PDF read the snapshot for frozen entries and the live shooter values otherwise. The CSV gains trailing `classification_at_match`, `class`, `club`, `country` columns; the `.xlsx` layout is unchanged.
- Entries cannot be added after a match leaves draft, so no late-entry freeze exists. Restoring from the score log creates unfrozen entries that fall back to live values.
