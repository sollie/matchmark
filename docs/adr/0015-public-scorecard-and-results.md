---
status: accepted
---

# Scorecard copies and Results are public and ignore visibility flags

A shooter should be able to keep a record of a finished match without an account, and the club wants to share results with a link. So the Scorecard copy (and the Results view and PDF that build on it) is public: no login, no device pairing and no token. The address is the competition id and the lane, both small sequential integers, so anyone can read the copy of any entry of any finished competition by counting.

- The spectator visibility flags do not apply to the Scorecard copy. The flags govern the live spectator screen while a match is running; a copy handed to the shooter after the match is a different audience and a different purpose. The Results view and the public Results PDF do honour the flags (#284, #286).
- Only a finished competition is readable. Before that the endpoint answers 404, the same as for an unknown competition, an empty lane or a withdrawn entry, so it does not reveal that a competition exists.
- The copy contains the shooter's name, WAID, category, lane, ring counts and the scorer's name per stage. A tokenised link would be shared in the same chat groups anyway, so a token adds friction without adding privacy.
- Nothing is cached by the server or by shared caches: responses carry `Cache-Control: private, no-cache`, and the SPA adds the data version to image URLs as a cache-buster because CDNs such as Cloudflare cache by file extension.

## Consequences

Sequential ids make the data enumerable. That is accepted; the cost of a scrape is bounded by the render protection in ADR 0016, and rate limiting beyond the application (per-IP edge limits) is left to the CDN or reverse proxy. A club that cannot accept public results should not expose the instance.
