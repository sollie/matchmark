package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ExportStageRow struct {
	StageNumber int
	Score       int
	XCount      int
	RingX       int
	Ring10      int
	Ring9       int
	Ring8       int
	Ring7       int
	Ring0       int
	RingM       int
}

type ExportShooterRow struct {
	ShooterID   int64
	Name        string
	WAID        string
	Club        *string
	Discipline  string
	WeaponClass string
	TotalScore  int
	XCount      int
	RingX       int
	Ring10      int
	Ring9       int
	Ring8       int
	Ring7       int
	Ring0       int
	RingM       int
	Stages      []ExportStageRow
}

type ExportRepository struct {
	db *sql.DB
}

func NewExportRepository(db *sql.DB) *ExportRepository {
	return &ExportRepository{db: db}
}

func (r *ExportRepository) GetCompetitionExport(ctx context.Context, competitionID int64) (_ []ExportShooterRow, retErr error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			sh.id,
			sh.name,
			sh.wa_id,
			sh.club,
			sh.discipline,
			sh.weapon_class,
			COALESCE(SUM(h.points), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'X'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'X'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '10' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '9'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '8'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '7'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '0'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'M'  THEN 1 ELSE 0 END), 0)
		FROM shooters sh
		LEFT JOIN stages st ON st.shooter_id = sh.id
		LEFT JOIN hits h ON h.stage_id = st.id
		WHERE sh.competition_id = ?
		GROUP BY sh.id
		ORDER BY sh.lane_number
	`, competitionID)
	if err != nil {
		return nil, fmt.Errorf("export query: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []ExportShooterRow
	for rows.Next() {
		var s ExportShooterRow
		if err := rows.Scan(
			&s.ShooterID, &s.Name, &s.WAID, &s.Club,
			&s.Discipline, &s.WeaponClass,
			&s.TotalScore, &s.XCount,
			&s.RingX, &s.Ring10, &s.Ring9, &s.Ring8, &s.Ring7, &s.Ring0, &s.RingM,
		); err != nil {
			return nil, fmt.Errorf("scan export row: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("export rows: %w", err)
	}

	for i := range out {
		stages, err := r.getShooterStages(ctx, out[i].ShooterID)
		if err != nil {
			return nil, err
		}
		out[i].Stages = stages
	}
	return out, nil
}

func (r *ExportRepository) getShooterStages(ctx context.Context, shooterID int64) (_ []ExportStageRow, retErr error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			st.stage_number,
			COALESCE(SUM(h.points), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'X'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'X'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '10' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '9'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '8'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '7'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = '0'  THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN h.ring = 'M'  THEN 1 ELSE 0 END), 0)
		FROM stages st
		LEFT JOIN hits h ON h.stage_id = st.id
		WHERE st.shooter_id = ?
		GROUP BY st.id
		ORDER BY st.stage_number
	`, shooterID)
	if err != nil {
		return nil, fmt.Errorf("stage export query: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []ExportStageRow
	for rows.Next() {
		var s ExportStageRow
		if err := rows.Scan(
			&s.StageNumber, &s.Score, &s.XCount,
			&s.RingX, &s.Ring10, &s.Ring9, &s.Ring8, &s.Ring7, &s.Ring0, &s.RingM,
		); err != nil {
			return nil, fmt.Errorf("scan stage export row: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
