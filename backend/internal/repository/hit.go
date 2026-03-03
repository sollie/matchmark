package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
)

type HitRepository struct {
	db *sql.DB
}

func NewHitRepository(db *sql.DB) *HitRepository {
	return &HitRepository{db: db}
}

func (r *HitRepository) ListByStage(ctx context.Context, stageID int64) (_ []model.Hit, retErr error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, stage_id, ring, points, created_at FROM hits WHERE stage_id = ? ORDER BY id`,
		stageID,
	)
	if err != nil {
		return nil, fmt.Errorf("list hits: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []model.Hit
	for rows.Next() {
		var h model.Hit
		if err := rows.Scan(&h.ID, &h.StageID, &h.Ring, &h.Points, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan hit: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *HitRepository) ReplaceForStage(ctx context.Context, stageID int64, hits []model.Hit) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM hits WHERE stage_id = ?`, stageID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete hits: %w", err)
	}

	for _, h := range hits {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO hits (stage_id, ring, points) VALUES (?, ?, ?)`,
			stageID, h.Ring, h.Points,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert hit: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *HitRepository) SumByShooter(ctx context.Context, shooterID int64) (score int, xCount int, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(h.points),0), COALESCE(SUM(CASE WHEN h.ring='X' THEN 1 ELSE 0 END),0)
		 FROM hits h
		 JOIN stages s ON s.id = h.stage_id
		 WHERE s.shooter_id = ?`,
		shooterID,
	)
	if err := row.Scan(&score, &xCount); err != nil {
		return 0, 0, fmt.Errorf("sum hits: %w", err)
	}
	return score, xCount, nil
}

func (r *HitRepository) SumByStage(ctx context.Context, stageID int64) (score int, xCount int, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(points),0), COALESCE(SUM(CASE WHEN ring='X' THEN 1 ELSE 0 END),0)
		 FROM hits WHERE stage_id = ?`,
		stageID,
	)
	if err := row.Scan(&score, &xCount); err != nil {
		return 0, 0, fmt.Errorf("sum stage hits: %w", err)
	}
	return score, xCount, nil
}
