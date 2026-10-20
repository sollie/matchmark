# MatchMark

Scoring for WA1500/PPC shooting competitions run by clubs, with results exported for delivery to a national representative.

## Language

**Club**:
An organisation that runs competitions and owns its own users, shooters and results.
_Avoid_: Tenant, organisation

**Competition**:
A single match event run by a club on a given day. Its name, date and course can be edited while it is a draft; the course is fixed once it has entries.
_Avoid_: Match, event

**Shooter**:
A person registered with the club who can enter competitions. A shooter with a WAID is a known shooter; one without is a new shooter.
_Avoid_: Competitor, participant

**WAID**:
The identifier the World Association assigns to a shooter and uses to match results to their record. New shooters have none yet.
_Avoid_: WA-ID, member number

**New shooter**:
A shooter without a WAID, typically fresh from the PPC course and needing a match to get the course accepted.
_Avoid_: Guest, beginner

**Entry**:
A shooter's participation in one competition, in one category, on one lane.
_Avoid_: Registration

**Category**:
The class a shooter competes in: Revolver, Pistol, Revolver OS, Pistol OS or Rifle. Each category is one sheet of the results report.
_Avoid_: Weapon class, class, division

**Lane**:
A numbered firing position on the range. At most one shooter occupies a lane.
_Avoid_: Position, bay

**Manned lane**:
A lane with a shooter in it for the current competition.
_Avoid_: Occupied lane, active lane

**Empty lane**:
A lane with no shooter. It is skipped when choosing whom to score.
_Avoid_: Unmanned lane, free lane

**Scorer**:
The shooter who records the hits of another shooter. Every scorer is a shooter, so a scorer's own lane is always manned.
_Avoid_: Marker, judge

**Score lane**:
The next manned lane after the scorer's own lane, wrapping around, whose shooter the scorer scores.
_Avoid_: Target lane, neighbour lane

**Discipline**:
A family of matches shot to the same rulebook course: 1500, 60 or 48. A discipline can have more than one course.
_Avoid_: Match type

**Course**:
The complete set of stages a competition is shot over, with a fixed total number of shots. A competition has exactly one course. The 5-shot revolver variant of the 48 discipline is its own course. Courses are keyed `150`, `60`, `48` and `40`, carry their UI label, and are served by `GET /api/rulebook`.
_Avoid_: Course of fire (as a noun for the whole set), program

**Stage**:
One course-of-fire segment shot by a shooter, with a fixed number of shots.
_Avoid_: String, round

**Hit**:
The recorded ring value of one shot.
_Avoid_: Shot result

**X count**:
The number of inner-ten (X) hits across an entry's stages. It breaks ties in the ranking and is the spectator column the visibility setting of the same name shows or hides.

**Lock**:
The scorer's confirmation that a stage's hits are final.
_Avoid_: Submit, sign-off

**Results report**:
The workbook, one sheet per category, that a club emails to the country representative. It lists a total score per shooter, plus the new shooters.
_Avoid_: Export, results file

**Scorecard**:
The record of one entry's hits per ring for each match, with the scorer's and shooter's names.
_Avoid_: Score sheet

**Device**:
A phone or tablet paired to one lane for a competition and used to score that lane's score lane.
_Avoid_: Terminal, client

**Replication**:
Copying the club's data to a separate host so it can be snapshotted and restored if the primary host is lost. It does not involve devices and is unrelated to how a device submits hits.
_Avoid_: Sync, backup

**Score log**:
The JSON `stage_locked` and `stage_unlocked` lines the server writes to stdout whenever a stage is locked or its locked state changes. Each line is self-contained, so a competition's scored results can be rebuilt by taking the latest line per stage.
_Avoid_: Scoring log

**Restore**:
Rebuilding competitions, entries and locked stages in an empty database from the Score log with `matchmark restore --apply`. Unscored entries, devices and pairings are not restored.
_Avoid_: Backup, replication

**Live update**:
A change notice the server pushes for a competition to the spectator, admin and scorer screens, so they refresh without asking.
_Avoid_: Poll, refresh, push notification

## Accounts

**User**:
A person with a login who manages the club's competitions and data. A user is not a shooter and not a scorer.
_Avoid_: Account, operator, member

**Admin**:
A user who can also manage other users. Only an admin can be allowed to download a backup.
_Avoid_: Superuser, owner

**Deactivated user**:
A user who can no longer log in but whose history stays attributed to them. An admin can reactivate them.
_Avoid_: Deleted user, disabled user, banned user

**Backup download**:
A full copy of the database file, including password hashes, taken by an admin who has been allowed to. It is not Replication and not Restore.
_Avoid_: Export, dump

**Audit log**:
The record of who did what: every change by a user, plus logins, failed logins, report downloads and backup downloads. It is separate from the Score log, which records locked results.
_Avoid_: Activity log, history
