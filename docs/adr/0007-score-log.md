---
status: accepted
---

# Score log: locked stages written to stdout as JSON

If the database is lost and no replica exists, every scored competition is gone. Container logs are usually retained but held only request lines. Each time a stage is locked, the server now writes a self-contained JSON `info` line to stdout, the **Score log**, through an injected `*slog.Logger`. The line is written only after the database write succeeded, and a logging failure never fails the lock.

Each line has `msg` and `event` set to `stage_locked` or `stage_unlocked`, plus `schema_version` (1), `emitted_at` (UTC, fixed-width millisecond precision), `seq` (integer, strictly increasing per process), `competition` (`id`, `name`, `date`, `course_of_fire` with `key`, `discipline`, `total_shots` taken from the rulebook), `entry` (`id`, `lane`, `category`, `shooter` with `waid`, `first_name`, `last_name`, `club`), `stage` (`number`, `expected_shots`), `ring_counts` (seven ints: X, 10, 9, 8, 7, 0, miss), `locked_by` (lane number, or `"Admin"`), `scorer_name` and `locked_at`. Visibility flags are not logged. The entry's category is the only class field, because the data model has no separate weapon class.

Replay rule: the last line per (competition id, entry id, stage number) by (`emitted_at`, `seq`) wins, and entry and competition context come from the latest line. A final `stage_unlocked` means the stage is not scored. Emission and the lookups for it are serialised under one mutex and `seq` is assigned inside it, so line order matches database order for the same stage and equal timestamps cannot make the replay ambiguous. The score-log context is read with a context that ignores request cancellation, so a cancelled request does not drop the line after the database write. A `stage_unlocked` line is built from the stage as it was just before the unlock, so it keeps the lock's `locked_by`, `scorer_name` and `locked_at`.

An admin ring-count edit on a locked stage keeps it locked and takes it over: the database records `scorer_lane` NULL and `scorer_name` `"Admin"`, and the re-emitted `stage_locked` has `locked_by` `"Admin"`, a new `emitted_at` and `seq`. The original device lock is deliberately not preserved. `locked_at` is stored in `stages.locked_at` (migrations 011 and 012, which backfills locked stages that predate the column with the migration time) so re-emitted lines keep the lock time.

Editing a competition (name, date, course), an entry (category, lane) or a shooter (name, club) re-emits `stage_locked` for every affected locked stage, keeping `locked_by`, `scorer_name` and `locked_at` as currently recorded and setting a new `emitted_at` and `seq`. Unlocked stages emit nothing.

## Considered Options

- A dedicated log file or table: rejected, stdout is already shipped by container log collection and the destination stays injectable.
- Replication only: kept as the primary protection, the Score log is a last-resort source.

## Consequences

Known limits: entries with no locked stage are not recoverable, and lines contain personal data (WAID, names, club), so log retention must be treated accordingly. A restore tool that reads the log is out of scope here.
