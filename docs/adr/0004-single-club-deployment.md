# One deployment per club, no multi-tenancy

The code serves a single club. Hosting for several clubs is a plausible future, but a `club_id` on every table adds data-leak risk and complexity now. Each club runs its own instance; code avoids global assumptions that would block adding tenancy later.
