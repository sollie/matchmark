# MatchMark

Scoring for WA1500/PPC shooting competitions run by clubs, with results exported for delivery to a national representative.

## Language

**Club**:
An organisation that runs competitions and owns its own users, shooters and results.
_Avoid_: Tenant, organisation

**Competition**:
A single match event run by a club on a given day.
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

**Stage**:
One course-of-fire segment shot by a shooter, with a fixed number of shots.
_Avoid_: String, round

**Hit**:
The recorded ring value of one shot.
_Avoid_: Shot result

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
