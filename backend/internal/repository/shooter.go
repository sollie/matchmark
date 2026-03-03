package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
)

type ShooterRepository struct {
	db *sql.DB
}

func NewShooterRepository(db *sql.DB) *ShooterRepository {
	return &ShooterRepository{db: db}
}

const shooterCols = `id, competition_id, name, wa_id, club, lane_number, discipline, weapon_class`

func scanShooter(row *sql.Row) (*model.Shooter, error) {
	var s model.Shooter
	if err := row.Scan(&s.ID, &s.CompetitionID, &s.Name, &s.WAID, &s.Club, &s.LaneNumber, &s.Discipline, &s.WeaponClass); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan shooter: %w", err)
	}
	return &s, nil
}

func scanShooterRow(rows *sql.Rows) (model.Shooter, error) {
	var s model.Shooter
	if err := rows.Scan(&s.ID, &s.CompetitionID, &s.Name, &s.WAID, &s.Club, &s.LaneNumber, &s.Discipline, &s.WeaponClass); err != nil {
		return model.Shooter{}, fmt.Errorf("scan shooter: %w", err)
	}
	return s, nil
}

func (r *ShooterRepository) GetByID(ctx context.Context, id int64) (*model.Shooter, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+shooterCols+` FROM shooters WHERE id = ?`, id,
	)
	return scanShooter(row)
}

func (r *ShooterRepository) GetByLane(ctx context.Context, competitionID int64, lane int) (*model.Shooter, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+shooterCols+` FROM shooters WHERE competition_id = ? AND lane_number = ?`,
		competitionID, lane,
	)
	return scanShooter(row)
}

func (r *ShooterRepository) ListByCompetition(ctx context.Context, competitionID int64) (_ []model.Shooter, retErr error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+shooterCols+` FROM shooters WHERE competition_id = ? ORDER BY lane_number`,
		competitionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list shooters: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && retErr == nil {
			retErr = cerr
		}
	}()

	var out []model.Shooter
	for rows.Next() {
		s, err := scanShooterRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ShooterRepository) Create(ctx context.Context, competitionID int64, name, waID string, club *string, lane int, discipline, weaponClass string) (*model.Shooter, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO shooters (competition_id, name, wa_id, club, lane_number, discipline, weapon_class) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		competitionID, name, waID, club, lane, discipline, weaponClass,
	)
	if err != nil {
		return nil, fmt.Errorf("create shooter: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *ShooterRepository) Update(ctx context.Context, id int64, name, waID string, club *string, lane int, discipline, weaponClass string) (*model.Shooter, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE shooters SET name = ?, wa_id = ?, club = ?, lane_number = ?, discipline = ?, weapon_class = ? WHERE id = ?`,
		name, waID, club, lane, discipline, weaponClass, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update shooter: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *ShooterRepository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM shooters WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete shooter: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
