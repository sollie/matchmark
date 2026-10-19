package repository

import (
	"context"
	"strings"
)

func (r *Repository) BackupTo(ctx context.Context, path string) error {
	literal := "'" + strings.ReplaceAll(path, "'", "''") + "'"
	_, err := r.db.ExecContext(ctx, `VACUUM INTO `+literal)
	return err
}
