package service

import (
	"context"
	"os"
	"path/filepath"
)

func (s *Service) Backup(ctx context.Context, user User, dir string) (path string, cleanup func(), err error) {
	if !user.CanDownloadBackup {
		return "", nil, ErrBackupNotPermitted
	}
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
	s.audit.Record(ctx, AuditEvent{Action: "backup.download"})
	return path, cleanup, nil
}
