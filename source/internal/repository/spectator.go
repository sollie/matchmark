package repository

import "context"

type Visibility struct {
	Names       bool
	Clubs       bool
	Countries   bool
	StageScores bool
	RingCounts  bool
}

func AllVisible() Visibility {
	return Visibility{Names: true, Clubs: true, Countries: true, StageScores: true, RingCounts: true}
}

type SpectatorStage struct {
	Number int
	Counts RingCounts
	Status StageStatus
}

type SpectatorEntry struct {
	Entry
	LastName  string
	FirstName string
	Club      string
	Country   string
	Stages    []SpectatorStage
}

func (r *Repository) GetVisibility(ctx context.Context, competitionID int64) (Visibility, error) {
	var v Visibility
	err := r.db.QueryRowContext(ctx,
		`SELECT show_names, show_clubs, show_countries, show_stage_scores, show_ring_counts FROM competitions WHERE id = ?`,
		competitionID).Scan(&v.Names, &v.Clubs, &v.Countries, &v.StageScores, &v.RingCounts)
	return v, mapErr(err)
}

func (r *Repository) SetVisibility(ctx context.Context, competitionID int64, v Visibility) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE competitions SET show_names = ?, show_clubs = ?, show_countries = ?, show_stage_scores = ?, show_ring_counts = ? WHERE id = ?`,
		v.Names, v.Clubs, v.Countries, v.StageScores, v.RingCounts, competitionID)
	return affected(res, mapErr(err))
}

func (r *Repository) ListPublicCompetitions(ctx context.Context) ([]Competition, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, date, course, status FROM competitions WHERE status != 'draft' ORDER BY date DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	comps := []Competition{}
	for rows.Next() {
		var c Competition
		if err := rows.Scan(&c.ID, &c.Name, &c.Date, &c.Course, &c.Status); err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return comps, rows.Err()
}

func (r *Repository) ListSpectatorEntries(ctx context.Context, competitionID int64) ([]SpectatorEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT e.id, e.category, e.lane, e.withdrawn, sh.last_name, sh.first_name, sh.club, sh.country,
		        st.stage_number, st.x, st.ten, st.nine, st.eight, st.seven, st.zero, st.miss, st.status
		 FROM entries e
		 JOIN shooters sh ON sh.id = e.shooter_id
		 JOIN stages st ON st.entry_id = e.id
		 WHERE e.competition_id = ?
		 ORDER BY e.lane, e.id, st.stage_number`, competitionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	entries := []SpectatorEntry{}
	for rows.Next() {
		var e SpectatorEntry
		var s SpectatorStage
		if err := rows.Scan(&e.ID, &e.Category, &e.Lane, &e.Withdrawn, &e.LastName, &e.FirstName, &e.Club, &e.Country,
			&s.Number, &s.Counts.X, &s.Counts.Ten, &s.Counts.Nine, &s.Counts.Eight, &s.Counts.Seven, &s.Counts.Zero, &s.Counts.Miss, &s.Status); err != nil {
			return nil, err
		}
		if n := len(entries); n > 0 && entries[n-1].ID == e.ID {
			entries[n-1].Stages = append(entries[n-1].Stages, s)
			continue
		}
		e.CompetitionID = competitionID
		e.Stages = []SpectatorStage{s}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
