package service

import (
	"context"
	"database/sql"
	"fmt"
)

var ErrSpectatorNotAvailable = fmt.Errorf("competition not found or not available for public viewing")

type SpectatorCompetition struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

type SpectatorStage struct {
	StageNumber int  `json:"stage_number"`
	Score       int  `json:"score"`
	XCount      int  `json:"x_count"`
	Locked      bool `json:"locked"`
}

type SpectatorLane struct {
	LaneNumber       int              `json:"lane_number"`
	ShooterName      string           `json:"shooter_name"`
	Club             *string          `json:"club"`
	Discipline       string           `json:"discipline"`
	WeaponClass      string           `json:"weapon_class"`
	CumulativeScore  int              `json:"cumulative_score"`
	CumulativeXCount int              `json:"cumulative_x_count"`
	Stages           []SpectatorStage `json:"stages"`
}

type SpectatorSnapshot struct {
	CompetitionID   int64           `json:"competition_id"`
	CompetitionName string          `json:"competition_name"`
	Lanes           []SpectatorLane `json:"lanes"`
}

type SpectatorService struct {
	db *sql.DB
}

func NewSpectatorService(db *sql.DB) *SpectatorService {
	return &SpectatorService{db: db}
}

func (s *SpectatorService) GetActiveCompetitions(ctx context.Context) ([]SpectatorCompetition, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, date, status FROM competitions WHERE spectator_enabled = 1 AND status = 'active' ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list spectator competitions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []SpectatorCompetition
	for rows.Next() {
		var c SpectatorCompetition
		if err := rows.Scan(&c.ID, &c.Name, &c.Date, &c.Status); err != nil {
			return nil, fmt.Errorf("scan spectator competition: %w", err)
		}
		out = append(out, c)
	}
	if out == nil {
		out = []SpectatorCompetition{}
	}
	return out, rows.Err()
}

func (s *SpectatorService) GetSnapshot(ctx context.Context, competitionID int64) (*SpectatorSnapshot, error) {
	var name string
	var enabled, active int
	err := s.db.QueryRowContext(ctx,
		`SELECT name, spectator_enabled, CASE WHEN status = 'active' THEN 1 ELSE 0 END FROM competitions WHERE id = ?`,
		competitionID,
	).Scan(&name, &enabled, &active)
	if err != nil {
		return nil, ErrSpectatorNotAvailable
	}
	if enabled == 0 || active == 0 {
		return nil, ErrSpectatorNotAvailable
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			sh.lane_number,
			sh.name,
			sh.club,
			sh.discipline,
			sh.weapon_class,
			COALESCE(SUM(h.points), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'X' THEN 1 ELSE 0 END), 0)
		FROM shooters sh
		LEFT JOIN stages st ON st.shooter_id = sh.id
		LEFT JOIN hits h ON h.stage_id = st.id
		WHERE sh.competition_id = ?
		GROUP BY sh.id
		ORDER BY SUM(h.points) DESC, SUM(CASE WHEN h.ring = 'X' THEN 1 ELSE 0 END) DESC
	`, competitionID)
	if err != nil {
		return nil, fmt.Errorf("query spectator lanes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type laneRow struct {
		laneNumber      int
		shooterName     string
		club            *string
		discipline      string
		weaponClass     string
		cumulativeScore int
		cumulativeX     int
	}

	var laneRows []laneRow
	for rows.Next() {
		var lr laneRow
		if err := rows.Scan(
			&lr.laneNumber, &lr.shooterName, &lr.club,
			&lr.discipline, &lr.weaponClass,
			&lr.cumulativeScore, &lr.cumulativeX,
		); err != nil {
			return nil, fmt.Errorf("scan spectator lane: %w", err)
		}
		laneRows = append(laneRows, lr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("spectator lane rows: %w", err)
	}

	lanes := make([]SpectatorLane, 0, len(laneRows))
	for _, lr := range laneRows {
		stages, err := s.getShooterStages(ctx, competitionID, lr.laneNumber)
		if err != nil {
			return nil, err
		}
		lanes = append(lanes, SpectatorLane{
			LaneNumber:       lr.laneNumber,
			ShooterName:      lr.shooterName,
			Club:             lr.club,
			Discipline:       lr.discipline,
			WeaponClass:      lr.weaponClass,
			CumulativeScore:  lr.cumulativeScore,
			CumulativeXCount: lr.cumulativeX,
			Stages:           stages,
		})
	}

	return &SpectatorSnapshot{
		CompetitionID:   competitionID,
		CompetitionName: name,
		Lanes:           lanes,
	}, nil
}

func (s *SpectatorService) getShooterStages(ctx context.Context, competitionID int64, laneNumber int) (_ []SpectatorStage, retErr error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			st.stage_number,
			COALESCE(SUM(h.points), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'X' THEN 1 ELSE 0 END), 0),
			st.fully_locked
		FROM stages st
		JOIN shooters sh ON sh.id = st.shooter_id
		LEFT JOIN hits h ON h.stage_id = st.id
		WHERE sh.competition_id = ? AND sh.lane_number = ?
		GROUP BY st.id
		ORDER BY st.stage_number
	`, competitionID, laneNumber)
	if err != nil {
		return nil, fmt.Errorf("query spectator stages: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []SpectatorStage
	for rows.Next() {
		var ss SpectatorStage
		var locked int
		if err := rows.Scan(&ss.StageNumber, &ss.Score, &ss.XCount, &locked); err != nil {
			return nil, fmt.Errorf("scan spectator stage: %w", err)
		}
		ss.Locked = locked == 1
		out = append(out, ss)
	}
	if out == nil {
		out = []SpectatorStage{}
	}
	return out, rows.Err()
}
