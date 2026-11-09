---
status: accepted
---

# Shooter classification from the WA1500 register

Results and the shooter page show each shooter's WA1500 Classification. The register page already used for lookups (ADR 0014) has one column per Category (`class. Revolver`, `class. Pistol`, `class. Revolver OS`, `class. Pistol OS`, `class. Carbine`).

- Stored as five nullable integer columns on `shooters` (`class_revolver`, `class_pistol`, `class_revolver_os`, `class_pistol_os`, `class_carbine`), each `CHECK (BETWEEN 0 AND 5)`, plus `wa1500_synced_at`. 0 is Unclassified, 1 to 5 are Marksman, Sharpshooter, Expert, Master, High Master. NULL means not fetched. The names live in a code translation table; the API sends the integer and the name. No SQLite lookup table: the set is fixed and a join buys nothing.
- A refresh searches the register by WAID and uses only the row whose WAID matches exactly. It overwrites name, country, club, every Classification and the timestamp. A missing or unrecognised class column keeps the old value (the latter is logged). Failures and not-found results never touch the timestamp and are retried with a short backoff.
- Adding a shooter to a Match refreshes synchronously (about 3 s timeout, best effort) when the timestamp is older than 24 hours; failure never blocks the entry. A background goroutine ticks hourly, skips archived shooters, and refreshes shooters with a WAID whose timestamp is NULL or older than 7 days, paced at about one request per 2 s.
- Shooters without a WAID have no Classification; the Shooter merge handles them once they get a WAID.
- Results show a "Class" column with the Classification for the entry's Category, hidden for NULL and Unclassified. The shooter page and admin list show one chip per Category. Reports are unchanged.

## Consequences

Classification and club were live values here; ADR 0025 snapshots them per entry when a match goes active. Refreshes now overwrite local name, country and club in the background, so local edits to a shooter with a WAID do not survive a refresh.
