package repository

import "context"

type Visibility struct {
	Names         bool
	Clubs         bool
	Countries     bool
	SegmentScores bool
	RingCounts    bool
}

type SpectatorSegment struct {
	Number int
	Counts RingCounts
	Status SegmentStatus
}

type SpectatorEntry struct {
	Entry
	LastName  string
	FirstName string
	Club      string
	Country   string
	Segments  []SpectatorSegment
}

func (r *Repository) GetVisibility(ctx context.Context, matchID int64) (Visibility, error) {
	var v Visibility
	err := r.db.QueryRowContext(ctx,
		`SELECT show_names, show_clubs, show_countries, show_segment_scores, show_ring_counts FROM matches WHERE id = ?`,
		matchID).Scan(&v.Names, &v.Clubs, &v.Countries, &v.SegmentScores, &v.RingCounts)
	return v, mapErr(err)
}

func (r *Repository) SetVisibility(ctx context.Context, matchID int64, v Visibility) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE matches SET show_names = ?, show_clubs = ?, show_countries = ?, show_segment_scores = ?, show_ring_counts = ? WHERE id = ?`,
		v.Names, v.Clubs, v.Countries, v.SegmentScores, v.RingCounts, matchID)
	return affected(res, mapErr(err))
}

func (r *Repository) ListPublicMatches(ctx context.Context) ([]Match, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, date, course, status FROM matches WHERE status != 'draft' ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	comps := []Match{}
	for rows.Next() {
		var c Match
		if err := rows.Scan(&c.ID, &c.Name, &c.Date, &c.Course, &c.Status); err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, rows.Err()
}

func (r *Repository) ListSpectatorEntries(ctx context.Context, matchID int64) ([]SpectatorEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.id, e.category, e.lane, e.withdrawn, sh.last_name, sh.first_name, sh.club, sh.country,
		        st.segment_number, st.x, st.ten, st.nine, st.eight, st.seven, st.zero, st.miss, st.status
		 FROM entries e
		 JOIN shooters sh ON sh.id = e.shooter_id
		 JOIN segments st ON st.entry_id = e.id
		 WHERE e.match_id = ?
		 ORDER BY e.lane, e.id, st.segment_number`, matchID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	entries := []SpectatorEntry{}
	for rows.Next() {
		var e SpectatorEntry
		var s SpectatorSegment
		if err := rows.Scan(&e.ID, &e.Category, &e.Lane, &e.Withdrawn, &e.LastName, &e.FirstName, &e.Club, &e.Country,
			&s.Number, &s.Counts.X, &s.Counts.Ten, &s.Counts.Nine, &s.Counts.Eight, &s.Counts.Seven, &s.Counts.Zero, &s.Counts.Miss, &s.Status); err != nil {
			return nil, err
		}
		if n := len(entries); n > 0 && entries[n-1].ID == e.ID {
			entries[n-1].Segments = append(entries[n-1].Segments, s)
			continue
		}
		e.MatchID = matchID
		e.Segments = []SpectatorSegment{s}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
