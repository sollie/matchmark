package repository

import "context"

func (r *Repository) BackupTo(ctx context.Context, path string) error {
	_, err := r.db.ExecContext(ctx, `VACUUM INTO ?`, path)
	return err
}
