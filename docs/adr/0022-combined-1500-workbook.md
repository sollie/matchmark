---
status: accepted
---

# Combined 1500 workbook

The country representative only needs the 1500 results, and the per-Match workbook zip from ADR 0016 makes the club send several files. A Competition now has one **Combined 1500 workbook** (`.xlsx`) holding every finished Match whose course has the 1500 discipline; 48, 60 and 40 Matches never go in, and unfinished 1500 Matches are skipped. CSV and PDF downloads stay zipped per Match and cover all Matches, and the per-Match xlsx zip stays.

The workbook is filled from the same embedded template as the per-Match report, with one sheet per category and one known-shooters block and one new-shooters block per sheet, rows added as needed. `A1` holds the Competition start date and `B1` the Competition name instead of a Match name and date. Entries come from the same selection, totals and sort order as the per-Match report plus a Match ID tiebreak; there is no second scoring path. A shooter who shot several of the Matches gets one row per Match, with the same WAID and each Match's own total. Repeat rows are not merged or summed, and nothing marks which Match a row belongs to.

Alternatives rejected:

- One block per Match stacked inside each sheet: keeps Match identity, but repeats the template layout per Match and the receiving side reads the template by position.
- One flat list with a Match column: changes the template's columns.
- Summing a shooter's totals over the Matches: hides the individual results and gives a number no rulebook defines.

The rules differ on repeat entries: the Norwegian federation allows several Matches of the same category on one day, the international federation does not. MatchMark enforces neither for now, and a human imposes any restriction at signup (see the follow-up validation issue and its planned config toggle). Duplicate WAIDs in the workbook are therefore expected.

With no finished 1500 Match the endpoint answers 422 and the admin UI hides the download. Unfinished 1500 Matches are not listed in the workbook; the UI warns before the download instead.
