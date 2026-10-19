package service

import (
	"context"
	"os"
	"path/filepath"
)

func (s *Service) Backup(ctx context.Context, dir string) (path string, cleanup func(), err error) {
	tmp, err := os.MkdirTemp(dir, "matchmark-backup-")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(tmp) }
	path = filepath.Join(tmp, "matchmark.db")
	if err := s.repo.BackupTo(ctx, path); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}
