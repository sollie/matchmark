package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
)

type LockRepository struct {
	db *sql.DB
}

func NewLockRepository(db *sql.DB) *LockRepository {
	return &LockRepository{db: db}
}

func (r *LockRepository) GetByStageAndUser(ctx context.Context, stageID, userID int64) (*model.StageLock, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, stage_id, user_id, locked, locked_time FROM stage_locks WHERE stage_id = ? AND user_id = ?`,
		stageID, userID,
	)
	var l model.StageLock
	var locked int
	if err := row.Scan(&l.ID, &l.StageID, &l.UserID, &locked, &l.LockedTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan stage lock: %w", err)
	}
	l.Locked = locked == 1
	return &l, nil
}

func (r *LockRepository) Upsert(ctx context.Context, stageID, userID int64, locked bool) error {
	if locked {
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO stage_locks (stage_id, user_id, locked, locked_time)
			 VALUES (?, ?, 1, strftime('%Y-%m-%dT%H:%M:%SZ','now'))
			 ON CONFLICT(stage_id, user_id) DO UPDATE SET locked=1, locked_time=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
			stageID, userID,
		)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO stage_locks (stage_id, user_id, locked, locked_time)
		 VALUES (?, ?, 0, NULL)
		 ON CONFLICT(stage_id, user_id) DO UPDATE SET locked=0, locked_time=NULL`,
		stageID, userID,
	)
	return err
}

func (r *LockRepository) IsLockedByScorer(ctx context.Context, stageID int64) (bool, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stage_locks WHERE stage_id = ? AND locked = 1`,
		stageID,
	)
	var n int
	if err := row.Scan(&n); err != nil {
		return false, fmt.Errorf("check scorer lock: %w", err)
	}
	return n > 0, nil
}

func (r *LockRepository) ClearForStage(ctx context.Context, stageID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE stage_locks SET locked = 0, locked_time = NULL WHERE stage_id = ?`,
		stageID,
	)
	return err
}

func (r *LockRepository) SetFullyLocked(ctx context.Context, stageID int64, locked bool) error {
	v := 0
	if locked {
		v = 1
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE stages SET fully_locked = ? WHERE id = ?`,
		v, stageID,
	)
	return err
}
