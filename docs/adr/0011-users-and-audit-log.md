---
status: proposed
---

# Named users with roles replace the single admin; every action is audited

One shared admin password cannot say who changed what, and several people now manage competitions. Each person gets a **User** with a username and password. Roles are `admin` and `user`; only admins manage users. Admins carry a `can_download_backup` flag, since a backup holds every password hash; regular users cannot have it. Users are deactivated and can be reactivated, never deleted, so history keeps its meaning.

The first admin is seeded from `ADMIN_USERNAME` and `ADMIN_PASSWORD` when the users table is empty; the variables are ignored afterwards. `matchmark reset-password <username>` is the lockout recovery path. The last active admin cannot be demoted or deactivated.

Passwords are hashed with argon2id, a random salt per user, and stored as PHC strings; weaker stored parameters are upgraded at login. Sessions are per user and are revoked on deactivation, demotion or password change. New and reset passwords must be changed at next login.

Every mutation, login, failed login, report download and backup download is written to an `audit_log` table and to a stdout `audit` JSON line. The Score log's `locked_by` carries the username for user locks instead of the literal `"Admin"`. There are no live installs, so none of this is backward compatible.

## Considered Options

- Keep one admin and add a log: rejected, it cannot attribute actions.
- Per-user permission flags beyond backup: rejected for now, two roles are enough.
- Hard-delete users: rejected, it would orphan audit rows.

## Consequences

ADR 0001's "the admin is the only account" no longer holds; scorers are still devices. A backup is as sensitive as the user table, so downloads are audited. Restore rebuilds from the Score log only, so users come back via seeding.
