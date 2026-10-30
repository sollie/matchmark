# MatchMark

Scoring for WA1500/PPC shooting competitions run by clubs, with results exported for delivery to a national representative.

## Language

**Club**:
An organisation that runs competitions and owns its own users, shooters and results.
_Avoid_: Tenant, organisation

**Competition**:
An announced event of one or more days run by a club, holding one or more Matches. It has a name and a start and end date and no status of its own.
_Avoid_: Tournament, event

**Match**:
One course shot by the shooters of a Competition, on one date inside the Competition's range. It has a name, a course, a status (draft, active, finished) and its own entries, lanes, stages and results. Its name, date and course can be edited while it is a draft; the course is fixed once it has entries. A shooter has at most one entry per Match.
_Avoid_: Competition (for a single course), event

**Back to draft**:
The admin action that returns an active Match to draft, only while no hit counts are stored for any entry (including unlocked and withdrawn ones). It does not restart the Match: pairings, Screen assignments, lanes and the current stage are kept, devices show "not started" and spectators do not see the Match until it is started again, which sets a new start time.
_Avoid_: Reset, restart, reopen

**Part**:
One of the rulebook's "Match 1" to "Match 5" groups of stages within the 150 course, as the PDFs name them. A Part can span several Scoring segments (Match 5 has two). Segment names carry the Part name in the UI and on reports.
_Avoid_: Match (for a group of stages)

**Shooter**:
A person registered with the club who can enter Matches. A shooter with a WAID is a known shooter; one without is a new shooter. Admins find a shooter by searching the local registry on WAID or name. A shooter with any entry, including a withdrawn one, cannot be deleted.
_Avoid_: Competitor, participant

**Archived shooter**:
A shooter an admin has retired from the club. It stays in past results, reports and the audit trail but is hidden from pickers and cannot be entered into new Matches until unarchived. Its WAID stays reserved.
_Avoid_: Deleted shooter, inactive shooter

**Shooter merge**:
Absorbing a duplicate shooter into another one, moving all its entries to the target and then removing it. Refused when both have different WAIDs or both have an entry in the same Match.
_Avoid_: Dedupe, combine

**WAID**:
The identifier the World Association assigns to a shooter and uses to match results to their record. New shooters have none yet.
_Avoid_: WA-ID, member number

**Normalized name**:
A shooter's name in the one canonical form the registry stores: the last name in uppercase, the first name with every word capitalised (also after a hyphen or apostrophe), whitespace trimmed and collapsed to single spaces. Creating or updating a shooter with any other form is rejected; WA1500 import normalizes before saving.
_Avoid_: Cleaned name, formatted name

**WA1500 import**:
Copying a shooter found in the WA1500 register into the club's own shooters. A shooter with the same WAID is updated; one that only shares the name must be confirmed as the same person or created as a new shooter.
_Avoid_: Sync

**New shooter**:
A shooter without a WAID, typically fresh from the PPC course and needing a Match to get the course accepted.
_Avoid_: Guest, beginner

**Entry**:
A shooter's participation in one Match, in one category, on one lane.
_Avoid_: Registration

**Category**:
The class a shooter competes in: Revolver, Pistol, Revolver OS, Pistol OS or Carbine; the rulebook limits which ones a course allows. Each category is one sheet of the results report.
_Avoid_: Weapon class, class, division

**Carbine**:
The category for Carbine 1500, shot over the 150-shot course on reduced WA1500-29 targets at the same distances; ring values and scoring are unchanged. It is not the Rifle 1500 the rulebook reserves for a future match.
_Avoid_: Rifle

**Lane**:
A numbered firing position on the range. At most one shooter occupies a lane.
_Avoid_: Position, bay

**Manned lane**:
A lane with a shooter in it for the current Match.
_Avoid_: Occupied lane, active lane

**Empty lane**:
A lane with no shooter. It is skipped when choosing whom to score.
_Avoid_: Unmanned lane, free lane

**Scorer**:
The shooter who records the hits of another shooter. Every scorer is a shooter, so a scorer's own lane is always manned.
_Avoid_: Marker, judge

**Score lane**:
The next manned lane after the scorer's own lane, wrapping around, whose shooter the scorer scores. When an entry is withdrawn between segments, the scorers who scored it are rerouted to the next manned, non-withdrawn lane from the following segment on; accepted segments keep their original scorer.
_Avoid_: Target lane, neighbour lane

**Discipline**:
A family of Matches shot to the same rulebook course: 1500, 60 or 48. A discipline can have more than one course.
_Avoid_: Match type

**Course**:
The complete set of stages a Match is shot over, with a fixed total number of shots. A Match has exactly one course. The 5-shot revolver variant of the 48 discipline is its own course. Courses are keyed `150`, `60`, `48` and `40`, carry their UI label, and are served by `GET /api/rulebook`.
_Avoid_: Course of fire (as a noun for the whole set), program

**Stage**:
One distance-and-position block of a course, numbered in the rulebook. A descriptive rulebook term only: nothing is recorded, locked or accepted per Stage, and the scorer screen does not show stage distances.
_Avoid_: String, round

**Scoring segment**:
A run of Stages between two "Score the target" lines on the score card, with one set of seven ring counts that must sum to the segment's shots. The 150 course has six (Match 1, Match 2, Match 3, Match 4, Match 5 · Stage 1+2, Match 5 · Stage 3+4); the 48, 60 and 40 courses have two (Stage 1+2, Stage 3+4). Names, stage lists and shots live in `internal/rulebook` and are served by `GET /api/rulebook`. Where the score cards ("St. 1+2") and the cheat sheet (Match 1/Match 2) differ, the score-card names win.
_Avoid_: Stage (for what is scored), section

**Ring value**:
What a hit is worth for its ring: X and 10 are worth ten, 9 nine, 8 eight, 7 seven, and 0 and M nothing.
_Avoid_: Points, ring score

**Segment score**:
The sum of the ring values of a Scoring segment's hits. An entry's total is the sum of its segment scores.
_Avoid_: Points

**Segment state**:
What a spectator sees for one shooter's Scoring segment: not started, in progress, awaiting acceptance or accepted. A disputed segment reopens and shows as in progress. Only an accepted segment shows its score.
_Avoid_: Segment status, stage state

**Hit**:
The recorded ring value of one shot.
_Avoid_: Shot result

**X count**:
The number of inner-ten (X) hits across an entry's segments. It breaks ties in the ranking and is the spectator column the visibility setting of the same name shows or hides.

**Lock**:
The scorer's confirmation that a segment's hits are complete, which puts the segment awaiting acceptance. It is not final until the shooter accepts.
_Avoid_: Submit, sign-off

**Awaiting acceptance**:
The state of a locked segment whose shooter has not yet accepted or disputed it. Its hits are frozen.
_Avoid_: Pending, submitted

**Accepted**:
The final state of a segment: the shooter accepted the scorer's lock, or an admin accepted it or locked it directly. Only accepted segments count in standings, progress and reports.
_Avoid_: Approved, confirmed, locked (for the final state)

**Dispute**:
The shooter's refusal of a locked segment. The segment reopens with its hits kept, and the scorer re-scores it, usually with the shooter and a range officer. A shooter can dispute any segment, without a reason.
_Avoid_: Reject, appeal

**Current segment**:
The one Scoring segment every device scores at a given time. The admin starts the next segment once every manned, non-withdrawn lane has accepted the current one (the gate); accepting the last segment finishes the match. Stored as `current_segment`.
_Avoid_: Active stage, current stage, round

**Dispute history**:
The admin table of a match's segment events (locks, acceptances, disputes, accept-all and unlocks, with actor and time), filterable by scorer, shooter and segment, with lock and dispute counts per scorer.

**Control room**:
The admin view of an active match's current segment, with one row per manned lane (scorer, shooter, status, disputes, paired and a best-effort connected indicator) and the Start next segment, Accept all, Accept and Unlock actions. Connected means an open scorer stream and is never stored.

**Results report**:
The workbook, one sheet per category, that a club emails to the country representative. It lists a total score per shooter, plus the new shooters.
_Avoid_: Export, results file

**Combined 1500 workbook**:
The Results report for a whole Competition: every finished 1500-discipline Match of the Competition in one workbook, one sheet per category, under the Competition's name and start date. A shooter who shot several of those Matches has one row per Match.
_Avoid_: Competition report, merged report

**Scorecard**:
The record of one entry's hits per ring for each match, with the scorer's and shooter's names.
_Avoid_: Score sheet

**Scorecard copy**:
The Scorecard of one entry rendered as an image on demand once its match is finished, for the shooter to keep. It shows each segment's ring counts with that segment's scorer (the range admin for admin-scored segments) and counts only accepted segments. It is public, ignores spectator visibility flags, is addressed by match and lane, is never stored and answers 404 for an unfinished match, a withdrawn entry or an empty lane.
_Avoid_: Receipt, printout

**Results view**:
The printable, public page of a finished match's standings, chosen from a picker of finished matches and honouring spectator visibility flags. It is the source of the Results PDF and is separate from the live spectator screen.
_Avoid_: Results page, leaderboard

**Results PDF**:
The Results view rendered as a PDF on demand and never stored. The public one honours visibility flags; the admin one, downloaded from the match list, shows everything.
_Avoid_: Results report (that is the `.xlsx`)

**Club settings**:
The club's name and optional logo, edited by admins at `/admin/club`. The logo appears on the Results view, the Results PDF and the Scorecard copy.
_Avoid_: Branding, profile

**Device**:
A phone or tablet paired to one lane for a match and used to score that lane's score lane.
_Avoid_: Terminal, client

**Replication**:
Copying the club's data to a separate host so it can be snapshotted and restored if the primary host is lost. It does not involve devices and is unrelated to how a device submits hits.
_Avoid_: Sync, backup

**Score log**:
The JSON `segment_status` lines the server writes to stdout whenever a segment changes status. Each line is self-contained, so a match's scored results, including segments awaiting acceptance, can be rebuilt by taking the latest line per segment.
_Avoid_: Scoring log

**Restore**:
Rebuilding matches, entries and scored segments in an empty database from the Score log with `matchmark restore --apply`. Unscored entries, devices and pairings are not restored.
_Avoid_: Backup, replication

**Live update**:
A change notice the server pushes for a match to the spectator, admin and scorer screens, so they refresh without asking.
_Avoid_: Poll, refresh, push notification

**Change recorder**:
The module that every state-changing operation reports to. Given one change (action, target, affected match, and any stages beyond open whose Score log lines must be re-emitted), it writes the Audit log entry, re-emits the Score log lines, then publishes a Live update, in that fixed order. Failures in these steps are logged and never fail the request. Replication stays with the operation that needs it.
_Avoid_: Event bus, notifier, hook

**Screen**:
A spectator display that has opted in to being remote controlled and been approved by an admin. It has a name the admin chose and is shown as online or offline.
_Avoid_: Display, monitor, kiosk, client

**Screen lifecycle**:
A Screen is _pending_ after it registers and shows its code, until an admin approves and names it; it is then _approved_ and unassigned (showing the most recently started Match), and _assigned_ once an admin gives it a Match. The screen stream reports `approved` alongside `state` (`waiting` or `match`), so a pending Screen and an approved but unassigned Screen waiting for a Match can be told apart.
_Avoid_: Accepted (reserved for segments)

**Unmanaged screen**:
A spectator display that has not opted in. It shows the most recently started Match until the viewer picks another.
_Avoid_: Free screen, standalone screen

**Screen assignment**:
The Match an admin chose for one Screen, or for all Screens at once. Only Matches that are not finished can be assigned. A Screen with no assignment shows the most recently started Match.
_Avoid_: Routing, pinning, push

**Waiting screen**:
What a Screen shows while its assigned Match is still a draft: only a waiting message, with no name, date, course or participants. It switches to the Match when the Match starts.
_Avoid_: Placeholder, blank screen

## Accounts

**User**:
A person with a login who manages the club's matches and data. A user is not a shooter and not a scorer.
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

**Login throttle**:
The module that decides whether a login attempt is allowed: it limits failures per client, slows repeated failures against one username, and deduplicates failed-login Audit log entries. It never sleeps; it returns the delay to apply.

**Audit log**:
The record of who did what: every change by a user, plus logins, failed logins, failed password changes, report downloads, backup downloads and `reset-password` command runs. Counts entered and locks made by a paired device are recorded with the actor `device`; the command is recorded with the actor `cli`. It is append-only, kept forever and readable only by admins. It is separate from the Score log, which records segment status changes, and from the Dispute history.
_Avoid_: Activity log, history
