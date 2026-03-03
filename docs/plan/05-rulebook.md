# MatchMark — Rulebook Data

Source: WA1500 Rulebook Issue 2026-01-01 + spec-bundle.yaml

## Scoring Ring Values

| Ring | Points | Notes |
|------|--------|-------|
| X    | 10     | Tracked separately as X-count for tie-breaking |
| 10   | 10     | |
| 9    | 9      | |
| 8    | 8      | |
| 7    | 7      | |
| 0    | 0      | Intentional zero (e.g. penalty) |
| M    | 0      | Miss — bullet did not pass through target + backer |

Tie-breaking order (Section 16): most Xs → fewest misses → fewest lowest-value shots
(repeated descending) → shoot-off at 25 yd standing without support.

## Disciplines and Stage Definitions

### 1500-Shot Course (10 stages, 150 shots total)

| Stage | Match | Distance | Shots | Time (s) | Position |
|-------|-------|----------|-------|----------|----------|
| 1  | 1 | 7 yd  | 12 | 20  | Standing without support; revolvers DA |
| 2  | 1 | 15 yd | 12 | 20  | Standing without support; revolvers DA |
| 3  | 2 | 25 yd | 18 | 90  | 6 kneeling + 6 L-barricade + 6 R-barricade; revolvers DA |
| 4  | 3 | 50 yd | 24 | 165 | 6 sitting + 6 prone + 6 L-barricade + 6 R-barricade; SA or DA |
| 5  | 4 | 25 yd | 12 | 35  | Standing without support |
| 6  | 4 | 25 yd | 12 | 35  | Standing without support (repeat) |
| 7  | 5 | 7 yd  | 12 | 20  | Standing without support; revolvers DA |
| 8  | 5 | 25 yd | 18 | 90  | 6 kneeling + 6 L-barricade + 6 R-barricade; revolvers DA |
| 9  | 5 | 50 yd | 24 | 165 | 6 sitting + 6 prone + 6 L-barricade + 6 R-barricade; SA or DA |
| 10 | 5 | 25 yd | 6  | 12  | Standing without support |

PDF subtotal groups: Match 1 (st.1–2), Match 2 (st.3), Match 3 (st.4),
Match 4 (st.5–6), Match 5 (st.7–10), Grand Total.

### 60-Shot Course (4 stages, 60 shots total)

| Stage | Distance | Shots | Time (s) | Position |
|-------|----------|-------|----------|----------|
| 1 | 7 yd  | 12 | 20  | Standing without support |
| 2 | 25 yd | 18 | 90  | 6 kneeling + 6 L-barricade + 6 R-barricade |
| 3 | 50 yd | 24 | 165 | 6 sitting + 6 prone + 6 L-barricade + 6 R-barricade |
| 4 | 25 yd | 6  | 12  | Standing without support |

PDF subtotal groups: Stages 1–2 (subtotal), Stages 3–4 (subtotal), Grand Total.

### 48-Shot Course (4 stages, 48 shots total; 5-shot variant: 40 shots)

| Stage | Distance | Shots (6-shot) | Shots (5-shot) | Time (s) | Position |
|-------|----------|----------------|----------------|----------|----------|
| 1 | 3 yd  | 6  | 5  | 8  | Standing single-handed; revolvers DA |
| 2 | 7 yd  | 12 | 10 | 20 | Standing without support; revolvers DA |
| 3 | 15 yd | 12 | 10 | 20 | Standing without support; revolvers DA |
| 4 | 25 yd | 18 | 15 | 90 | 6 (5) kneeling + 6 (5) L-barricade + 6 (5) R-barricade; revolvers DA |

5-shot variant auto-detected when `weapon_class == "Standard Revolver 2.75\" 5-Shot"`.
PDF subtotal groups: Stages 1–2 (subtotal), Stages 3–4 (subtotal), Grand Total.

## Weapon Classes per Discipline

### Discipline: 1500
- Revolver 1500
- Pistol 1500
- Revolver Optical Sight 1500
- Pistol Optical Sight 1500

### Discipline: 60
- Distinguished Revolver
- Distinguished Pistol
- Open Match

### Discipline: 48
- Standard Revolver 4.25"
- Standard Revolver 2.75"
- Standard Revolver 2.75" 5-Shot  ← triggers 40-shot variant
- Standard Semi-Automatic Pistol 5.5" Production
- Standard Semi-Automatic Pistol 5.5"
- Standard Semi-Automatic Pistol 3.7"

## Classification Thresholds (150-shot / 1500 points)

| Class | Score Range |
|-------|-------------|
| High Master | 1476–1500 |
| Master | 1440–1475 |
| Expert | 1379–1439 |
| Sharpshooter | 1290–1378 |
| Marksman | 0–1289 |

Classification requires 300 reported shots. Not enforced in MatchMark v1 — displayed
in export for reference only.

## Scoring Validation Rules (enforced by backend service layer)

- `ring` must be one of: `X`, `10`, `9`, `8`, `7`, `0`, `M`
- Number of hits submitted must equal `expected_shots` for the stage
- `expected_shots` is derived from discipline + weapon_class at shooter creation time
  and stored in the `stages` table — the source of truth at score time
- No negative points; no fractional values

## Lane-Based Peer Scoring Rule

```
scoreLane = ownLane - 1
if ownLane == 1:
    scoreLane = lanes_total
```

Example: 21 lanes, user on lane 1 → scores lane 21.
Example: 21 lanes, user on lane 8 → scores lane 7.

This enforces the competition convention: each shooter scores the shooter to their left;
lane 1 wraps to score the last lane.
