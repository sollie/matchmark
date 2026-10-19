---
status: accepted
---

# Score log: locked stages written to stdout as JSON

If the database is lost and no replica exists, every scored competition is gone. Container logs are usually retained but held only request lines. Each time a stage is locked, the server now writes a self-contained JSON `info` line to stdout, the **Score log**, through an injected `*slog.Logger`. The line is written only after the database write succeeded, and a logging failure never fails the lock.

Each line has `msg` and `event` set to `stage_locked` (later also `stage_unlocked`), plus `schema_version` (1), `emitted_at` (RFC 3339 UTC), `competition` (`id`, `name`, `date`, `course_of_fire` with `key`, `discipline`, `total_shots` taken from the rulebook), `entry` (`id`, `lane`, `category`, `shooter` with `waid`, `first_name`, `last_name`, `club`), `stage` (`number`, `expected_shots`), `ring_counts` (seven ints: X, 10, 9, 8, 7, 0, miss), `locked_by` (lane number, or `"Admin"`), `scorer_name` and `locked_at`. Visibility flags are not logged. The entry's category is the only class field, because the data model has no separate weapon class.

Replay rule: the last line per (competition id, entry id, stage number) by `emitted_at` wins, and entry and competition context come from the latest line. A final `stage_unlocked` means the stage is not scored. An admin ring-count edit on a locked stage keeps it locked and re-emits `stage_locked` with `locked_by` `"Admin"` and a new `emitted_at`. `locked_at` is stored in `stages.locked_at` (migration 011) so re-emitted lines keep the original lock time.

Editing a competition (name, date, course), an entry (category, lane) or a shooter (name, club) re-emits `stage_locked` for every affected locked stage, keeping `locked_by`, `scorer_name` and `locked_at` and setting a new `emitted_at`. Unlocked stages emit nothing.

## Considered Options

- A dedicated log file or table: rejected, stdout is already shipped by container log collection and the destination stays injectable.
- Replication only: kept as the primary protection, the Score log is a last-resort source.

## Consequences

Known limits: entries with no locked stage are not recoverable, and lines contain personal data (WAID, names, club), so log retention must be treated accordingly. A restore tool that reads the log is out of scope here.
