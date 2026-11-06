package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/sollie/matchmark/internal/rulebook"
)

func classColumn(category string) string {
	return "class_" + strings.ReplaceAll(strings.ToLower(category), " ", "_")
}

var classColumns = func() string {
	cols := make([]string, 0, len(rulebook.Categories()))
	for _, c := range rulebook.Categories() {
		cols = append(cols, classColumn(c))
	}
	return strings.Join(cols, ", ")
}()

type classScan struct {
	cells []sql.NullInt64
}

func (c *classScan) dest() []any {
	c.cells = make([]sql.NullInt64, len(rulebook.Categories()))
	out := make([]any, len(c.cells))
	for i := range c.cells {
		out[i] = &c.cells[i]
	}
	return out
}

func (c *classScan) values() map[string]int {
	var out map[string]int
	for i, category := range rulebook.Categories() {
		if !c.cells[i].Valid {
			continue
		}
		if out == nil {
			out = map[string]int{}
		}
		out[category] = int(c.cells[i].Int64)
	}
	return out
}

type ShooterRefresh struct {
	LastName  string
	FirstName string
	Country   string
	Club      string
	Classes   map[string]int
	SyncedAt  string
}

func (r *Repository) RefreshShooter(ctx context.Context, id int64, in ShooterRefresh) error {
	sets := []string{"last_name = ?", "first_name = ?", "country = ?", "club = ?", "wa1500_synced_at = ?"}
	args := []any{in.LastName, in.FirstName, in.Country, in.Club, in.SyncedAt}
	for _, category := range rulebook.Categories() {
		col := classColumn(category)
		sets = append(sets, col+" = COALESCE(?, "+col+")")
		if v, ok := in.Classes[category]; ok {
			args = append(args, v)
		} else {
			args = append(args, nil)
		}
	}
	res, err := r.db.ExecContext(ctx, `UPDATE shooters SET `+strings.Join(sets, ", ")+` WHERE id = ?`, append(args, id)...)
	return affected(res, mapErr(err))
}

func (r *Repository) ListShootersDueForSync(ctx context.Context, syncedBefore string) ([]Shooter, error) {
	return r.queryShooters(ctx, nil, 0,
		`WHERE archived = 0 AND waid IS NOT NULL AND (wa1500_synced_at IS NULL OR wa1500_synced_at < ?)`,
		"wa1500_synced_at IS NOT NULL, wa1500_synced_at, id", syncedBefore)
}
