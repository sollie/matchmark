package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
)

type AssignmentRepository struct {
	db *sql.DB
}

func NewAssignmentRepository(db *sql.DB) *AssignmentRepository {
	return &AssignmentRepository{db: db}
}

const assignmentJoin = `
	SELECT sa.id, sa.competition_id, sa.shooter_id, sa.user_id,
	       u.username, s.name, s.lane_number, sa.assigned_at
	FROM scorer_assignments sa
	JOIN users u ON u.id = sa.user_id
	JOIN shooters s ON s.id = sa.shooter_id`

func scanAssignment(row *sql.Row) (*model.ScorerAssignment, error) {
	var a model.ScorerAssignment
	if err := row.Scan(&a.ID, &a.CompetitionID, &a.ShooterID, &a.UserID, &a.Username, &a.ShooterName, &a.LaneNumber, &a.AssignedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan assignment: %w", err)
	}
	return &a, nil
}

func (r *AssignmentRepository) GetByUser(ctx context.Context, competitionID, userID int64) (*model.ScorerAssignment, error) {
	row := r.db.QueryRowContext(ctx,
		assignmentJoin+` WHERE sa.competition_id = ? AND sa.user_id = ?`,
		competitionID, userID,
	)
	return scanAssignment(row)
}

func (r *AssignmentRepository) GetByShooter(ctx context.Context, competitionID, shooterID int64) (*model.ScorerAssignment, error) {
	row := r.db.QueryRowContext(ctx,
		assignmentJoin+` WHERE sa.competition_id = ? AND sa.shooter_id = ?`,
		competitionID, shooterID,
	)
	return scanAssignment(row)
}

func (r *AssignmentRepository) Create(ctx context.Context, competitionID, shooterID, userID int64) (*model.ScorerAssignment, error) {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO scorer_assignments (competition_id, shooter_id, user_id) VALUES (?, ?, ?)`,
		competitionID, shooterID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("create assignment: %w", err)
	}
	return r.GetByUser(ctx, competitionID, userID)
}

func (r *AssignmentRepository) DeleteByShooter(ctx context.Context, competitionID, shooterID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM scorer_assignments WHERE competition_id = ? AND shooter_id = ?`,
		competitionID, shooterID,
	)
	return err
}

func (r *AssignmentRepository) DeleteByUser(ctx context.Context, competitionID, userID int64) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM scorer_assignments WHERE competition_id = ? AND user_id = ?`,
		competitionID, userID,
	)
	return err
}

func (r *AssignmentRepository) ListByCompetition(ctx context.Context, competitionID int64) (_ []model.ScorerAssignment, retErr error) {
	rows, err := r.db.QueryContext(ctx,
		assignmentJoin+` WHERE sa.competition_id = ? ORDER BY s.lane_number`,
		competitionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []model.ScorerAssignment
	for rows.Next() {
		var a model.ScorerAssignment
		if err := rows.Scan(&a.ID, &a.CompetitionID, &a.ShooterID, &a.UserID, &a.Username, &a.ShooterName, &a.LaneNumber, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
