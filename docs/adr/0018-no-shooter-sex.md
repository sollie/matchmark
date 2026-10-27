---
status: accepted
---

# Shooters carry no sex

Sex was stored on shooters only to fill the sex column of the `New shooters` block in the xlsx results report. It is not part of the rulebook or scoring, and neither the WA1500 search nor the Score log carries it, so the WA1500 import forced the admin to pick M or F and restore invented `M`, both only to satisfy a `NOT NULL CHECK`.

We drop it entirely. The report template keeps its column and the cell is left empty; the club representative who receives the results fills it in. Nothing is deployed, so `shooters.sex` is removed by editing `001_schema.sql` in place (the precedent of ADR 0017) and local databases must be recreated. A stale `sex` in a request body is ignored because unknown JSON fields are not rejected.

This supersedes the sex choice in ADR 0014 and the restored sex in ADR 0007.
