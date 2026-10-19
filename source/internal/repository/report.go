package repository

import "context"

type ReportStage struct {
	Number     int
	Counts     RingCounts
	Locked     bool
	ScorerLane *int
	ScorerName string
}

type ReportEntry struct {
	Entry
	Shooter Shooter
	Stages  []ReportStage
}

func (r *Repository) ReportEntries(ctx context.Context, competitionID int64) ([]ReportEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.id, e.category, e.lane, e.withdrawn,
		        sh.id, sh.last_name, sh.first_name, sh.sex, sh.country, sh.club, COALESCE(sh.waid, ''),
		        s.stage_number, s.x, s.ten, s.nine, s.eight, s.seven, s.zero, s.miss, s.locked, s.scorer_lane, COALESCE(s.scorer_name, '')
		 FROM entries e
		 JOIN shooters sh ON sh.id = e.shooter_id
		 JOIN stages s ON s.entry_id = e.id
		 WHERE e.competition_id = ?
		 ORDER BY sh.last_name, sh.first_name, e.id, s.stage_number`, competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	entries := []ReportEntry{}
	for rows.Next() {
		var e Entry
		var sh Shooter
		var st ReportStage
		var withdrawn, locked int
		if err := rows.Scan(&e.ID, &e.Category, &e.Lane, &withdrawn,
			&sh.ID, &sh.LastName, &sh.FirstName, &sh.Sex, &sh.Country, &sh.Club, &sh.WAID,
			&st.Number, &st.Counts.X, &st.Counts.Ten, &st.Counts.Nine, &st.Counts.Eight, &st.Counts.Seven, &st.Counts.Zero, &st.Counts.Miss,
			&locked, &st.ScorerLane, &st.ScorerName); err != nil {
			return nil, err
		}
		st.Locked = locked == 1
		if n := len(entries); n == 0 || entries[n-1].ID != e.ID {
			e.CompetitionID = competitionID
			e.ShooterID = sh.ID
			e.Withdrawn = withdrawn == 1
			entries = append(entries, ReportEntry{Entry: e, Shooter: sh})
		}
		last := &entries[len(entries)-1]
		last.Stages = append(last.Stages, st)
	}
	return entries, rows.Err()
}
