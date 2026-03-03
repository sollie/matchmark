package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type MonitorLane struct {
	LaneNumber       int     `json:"lane_number"`
	ShooterName      string  `json:"shooter_name"`
	Discipline       string  `json:"discipline"`
	WeaponClass      string  `json:"weapon_class"`
	CurrentStage     int     `json:"current_stage"`
	StageFullyLocked bool    `json:"stage_fully_locked"`
	ScorerLocked     bool    `json:"scorer_locked"`
	CumulativeScore  int     `json:"cumulative_score"`
	XCount           int     `json:"x_count"`
	AssignedUsername *string `json:"assigned_username"`
}

type MonitorSnapshot struct {
	CompetitionID   int64         `json:"competition_id"`
	CompetitionName string        `json:"competition_name"`
	Status          string        `json:"status"`
	Lanes           []MonitorLane `json:"lanes"`
}

type MonitorService struct {
	db           *sql.DB
	competitions *CompetitionService
}

func NewMonitorService(db *sql.DB, competitions *CompetitionService) *MonitorService {
	return &MonitorService{db: db, competitions: competitions}
}

func (s *MonitorService) GetSnapshot(ctx context.Context, competitionID int64) (*MonitorSnapshot, error) {
	comp, err := s.competitions.GetByID(ctx, competitionID)
	if err != nil {
		if errors.Is(err, ErrCompetitionNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			sh.lane_number,
			sh.name,
			sh.discipline,
			sh.weapon_class,
			COALESCE((
				SELECT st.stage_number
				FROM stages st
				WHERE st.shooter_id = sh.id AND st.fully_locked = 0
				ORDER BY st.stage_number
				LIMIT 1
			), (
				SELECT COUNT(*) FROM stages WHERE shooter_id = sh.id
			) + 1) AS current_stage,
			COALESCE((
				SELECT st.fully_locked
				FROM stages st
				WHERE st.shooter_id = sh.id AND st.stage_number = (
					SELECT st2.stage_number
					FROM stages st2
					WHERE st2.shooter_id = sh.id AND st2.fully_locked = 0
					ORDER BY st2.stage_number
					LIMIT 1
				)
				LIMIT 1
			), 0) AS stage_fully_locked,
			COALESCE((
				SELECT MAX(sl.locked)
				FROM stage_locks sl
				JOIN stages st ON st.id = sl.stage_id
				WHERE st.shooter_id = sh.id AND sl.locked = 1
				  AND st.stage_number = (
					SELECT st3.stage_number
					FROM stages st3
					WHERE st3.shooter_id = sh.id AND st3.fully_locked = 0
					ORDER BY st3.stage_number
					LIMIT 1
				  )
			), 0) AS scorer_locked,
			COALESCE((
				SELECT SUM(h.points)
				FROM hits h
				JOIN stages st ON st.id = h.stage_id
				WHERE st.shooter_id = sh.id
			), 0) AS cumulative_score,
			COALESCE((
				SELECT SUM(CASE WHEN h.ring = 'X' THEN 1 ELSE 0 END)
				FROM hits h
				JOIN stages st ON st.id = h.stage_id
				WHERE st.shooter_id = sh.id
			), 0) AS x_count,
			(
				SELECT u.username
				FROM scorer_assignments sa
				JOIN users u ON u.id = sa.user_id
				WHERE sa.competition_id = sh.competition_id AND sa.shooter_id = sh.id
				LIMIT 1
			) AS assigned_username
		FROM shooters sh
		WHERE sh.competition_id = ?
		ORDER BY sh.lane_number
	`, competitionID)
	if err != nil {
		return nil, fmt.Errorf("query monitor: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var lanes []MonitorLane
	for rows.Next() {
		var l MonitorLane
		var fullyLocked, scorerLocked int
		if err := rows.Scan(
			&l.LaneNumber,
			&l.ShooterName,
			&l.Discipline,
			&l.WeaponClass,
			&l.CurrentStage,
			&fullyLocked,
			&scorerLocked,
			&l.CumulativeScore,
			&l.XCount,
			&l.AssignedUsername,
		); err != nil {
			return nil, fmt.Errorf("scan monitor row: %w", err)
		}
		l.StageFullyLocked = fullyLocked == 1
		l.ScorerLocked = scorerLocked == 1
		lanes = append(lanes, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("monitor rows: %w", err)
	}

	if lanes == nil {
		lanes = []MonitorLane{}
	}

	return &MonitorSnapshot{
		CompetitionID:   comp.ID,
		CompetitionName: comp.Name,
		Status:          comp.Status,
		Lanes:           lanes,
	}, nil
}
