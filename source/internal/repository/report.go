package repository

import "context"

type ReportSegment struct {
	Number     int
	Counts     RingCounts
	Status     SegmentStatus
	ScorerLane *int
	ScorerName string
}

type ReportEntry struct {
	Entry
	Shooter  Shooter
	Segments []ReportSegment
}

func (r *Repository) ReportEntries(ctx context.Context, matchID int64) ([]ReportEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.id, e.category, e.lane, e.withdrawn, e.classification_at_match, e.class, e.club, e.country,
		        sh.id, sh.last_name, sh.first_name, sh.country, sh.club, COALESCE(sh.waid, ''),
		        sh.class_revolver, sh.class_pistol, sh.class_revolver_os, sh.class_pistol_os, sh.class_carbine,
		        s.segment_number, s.x, s.ten, s.nine, s.eight, s.seven, s.zero, s.miss, s.status, s.scorer_lane, COALESCE(s.scorer_name, '')
		 FROM entries e
		 JOIN shooters sh ON sh.id = e.shooter_id
		 JOIN segments s ON s.entry_id = e.id
		 WHERE e.match_id = ?
		 ORDER BY sh.last_name, sh.first_name, e.id, s.segment_number`, matchID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	entries := []ReportEntry{}
	for rows.Next() {
		var e Entry
		var sh Shooter
		var st ReportSegment
		var withdrawn int
		var classes classScan
		dest := append([]any{&e.ID, &e.Category, &e.Lane, &withdrawn, &e.ClassificationAtMatch, &e.Class, &e.Club, &e.Country,
			&sh.ID, &sh.LastName, &sh.FirstName, &sh.Country, &sh.Club, &sh.WAID}, classes.dest()...)
		if err := rows.Scan(append(dest,
			&st.Number, &st.Counts.X, &st.Counts.Ten, &st.Counts.Nine, &st.Counts.Eight, &st.Counts.Seven, &st.Counts.Zero, &st.Counts.Miss,
			&st.Status, &st.ScorerLane, &st.ScorerName)...); err != nil {
			return nil, err
		}
		sh.Classes = classes.values()
		if n := len(entries); n == 0 || entries[n-1].ID != e.ID {
			e.MatchID = matchID
			e.ShooterID = sh.ID
			e.Withdrawn = withdrawn == 1
			entries = append(entries, ReportEntry{Entry: e, Shooter: sh})
		}
		last := &entries[len(entries)-1]
		last.Segments = append(last.Segments, st)
	}
	return entries, rows.Err()
}
