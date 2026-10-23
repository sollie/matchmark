---
status: accepted
---

# Look up shooters by scraping the WA1500 register

New shooters are mostly already in the WA1500 register, but WA1500 offers no API: `https://wa1500.org/class.php` takes a form POST (`action=show&Suche=<first name|last name|club|WAID>`) and returns an HTML page. MatchMark scrapes that page instead of making the admin type every shooter by hand.

- The host is fixed in code (no user-supplied URL, so no SSRF). Requests have a timeout and a response size cap; the Latin-1 body is decoded to UTF-8 and the result rows are parsed with `golang.org/x/net/html`. The scraper sits behind an interface so tests use fixtures and never touch the network.
- Lookups are explicit (a Search button), rate limited and briefly cached. An unreachable or unparsable upstream returns 502 and the admin falls back to manual entry. The register is a convenience, never a dependency: everything selected is stored locally.
- WA1500 provides no sex, so the admin chooses M or F on import.
- The WAID is the identity key. `POST /api/shooters/import` updates a shooter whose WAID matches, silently. Without a WAID match, local shooters with the same first and last name (case-insensitive) make the server answer 409 with the candidates; the client resends with `update_shooter_id` (validated server-side to be one of the name matches) or `create_new`. This keeps two people with the same name apart and lets a manually entered shooter without a WAID receive one.

## Consequences

The scraper depends on the page markup (`tr.ranking` rows with `data-title` cells) and breaks if WA1500 redesigns it; the 502 fallback keeps manual entry working. Imported data overwrites the local name, country, club and sex of the matched shooter.
