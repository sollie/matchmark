---
status: accepted
---

# Equipment categories from the rulebook

The five categories (Revolver, Pistol, Revolver OS, Pistol OS, Carbine) are replaced by the rulebook's equipment classes: rules 3.1 to 3.13 plus Carbine 1500 (Carbine rulebook 1.0), with the rulebook titles verbatim. The project is pre-1.0, so this is a breaking change.

- `internal/rulebook` holds one ordered list in rulebook order. `GET /api/rulebook`, the web UI, the results, the PDF, the CSV and the `.xlsx` sheet order all derive from it.
- Each category is shot over exactly one course. The rulebook has no explicit table, so this mapping is an interpretation of section 3 and section 7 (7.8 is 60 shots, 7.9 is 48 (40) shots) and awaits clarification from WA1500: 150 shots for 3.1 to 3.4 and Carbine 1500; 60 shots for Open (3.5), Distinguished Revolver (3.6) and Distinguished Pistol (3.7); 48 shots for 3.8, 3.9 and 3.11 to 3.13; 40 shots for Standard Revolver 2,75" 5 Shot (3.10). Rule 3.13 says 3,7" while 5.6(c) says 3,65"; 3.13 is followed.
- Only the five 1500 categories are classified by the WA1500 register. The five `shooters.class_*` columns are unchanged, the Classification maps are keyed by the new category names, and the register columns (`class. Revolver` and so on) are mapped to them in `internal/rulebook`. The other nine categories have no Classification and are never grouped by Class.
- The results `.xlsx` keeps its positional template (ADR 0022) and exports only the five 1500 categories. The CSV, the PDF and the Results view include every category.
- Migration 005 rebuilds `entries` with the new `category` CHECK and renames the old values in 150-shot matches. Entries in 60, 48 and 40-shot matches cannot be mapped unambiguously, so the migration fails loudly on them and the operator must re-enter them first. Because rebuilding `entries` would cascade into `segments` and `segment_events`, migrations now run on a dedicated connection with foreign keys off, as the SQLite table-rebuild procedure recommends.
