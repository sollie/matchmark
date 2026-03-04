package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
)

type StageRepository struct {
	db *sql.DB
}

func NewStageRepository(db *sql.DB) *StageRepository {
	return &StageRepository{db: db}
}

func (r *StageRepository) CreateBulk(ctx context.Context, shooterID int64, stages []model.Stage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	for _, s := range stages {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO stages (shooter_id, stage_number, expected_shots) VALUES (?, ?, ?)`,
			shooterID, s.StageNumber, s.ExpectedShots,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert stage %d: %w", s.StageNumber, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (r *StageRepository) GetByShooterAndNumber(ctx context.Context, shooterID int64, stageNumber int) (*model.Stage, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, shooter_id, stage_number, expected_shots, fully_locked FROM stages WHERE shooter_id = ? AND stage_number = ?`,
		shooterID, stageNumber,
	)
	var s model.Stage
	var locked int
	if err := row.Scan(&s.ID, &s.ShooterID, &s.StageNumber, &s.ExpectedShots, &locked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan stage: %w", err)
	}
	s.FullyLocked = locked == 1
	return &s, nil
}
