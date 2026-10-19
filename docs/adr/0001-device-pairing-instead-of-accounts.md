# Scorers use lane-paired devices, not accounts

Scorers are shooters at a single club, 2–20 per event, and the old JWT/refresh-token/user-account system was far heavier than that needs. Each manned lane gets a pairing link/QR; a device that opens it receives a cookie bound to that competition and lane and can only score that lane's score lane. The admin is the only account (one password) and can revoke, re-pair, or score any lane.

## Considered Options

- Per-user accounts with JWT and refresh tokens (the v1 design): rejected, password management for rotating shooters and the token machinery are not worth it.
- Shared per-competition join code where scorers pick their lane: rejected, a device could claim the wrong lane.

## Consequences

Pairing tokens expire when the competition finishes. There is no scorer password reset and no scorer identity beyond the lane.
