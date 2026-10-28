package service

import (
	"context"
	"errors"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
)

type ShooterImport struct {
	Shooter         repository.Shooter
	UpdateShooterID int64
	CreateNew       bool
}

type ShooterImportResult struct {
	ID      int64
	Created bool
}

type ArchivedShooterError struct {
	Shooter repository.Shooter
}

func (e *ArchivedShooterError) Error() string { return ErrShooterArchived.Error() }

func (e *ArchivedShooterError) Is(target error) bool { return target == ErrShooterArchived }

type NameConflictError struct {
	Candidates []repository.Shooter
}

func (e *NameConflictError) Error() string { return "shooter name matches existing shooters" }

func (e *NameConflictError) Is(target error) bool { return target == ErrConflict }

func (s *Service) ImportShooter(ctx context.Context, in ShooterImport) (ShooterImportResult, error) {
	sh := in.Shooter
	sh.FirstName, sh.LastName = NormalizeFirstName(sh.FirstName), NormalizeLastName(sh.LastName)
	sh.WAID = strings.TrimSpace(sh.WAID)
	if sh.WAID == "" || !validShooter(sh) || (in.CreateNew && in.UpdateShooterID != 0) {
		return ShooterImportResult{}, ErrInvalid
	}

	existing, err := s.repo.FindShooterByWAID(ctx, sh.WAID)
	switch {
	case err == nil:
		return s.importUpdate(ctx, existing, sh)
	case !errors.Is(err, repository.ErrNotFound):
		return ShooterImportResult{}, mapRepoErr(err)
	}

	if in.CreateNew {
		return s.importCreate(ctx, sh)
	}
	matches, err := s.repo.FindShootersByName(ctx, sh.FirstName, sh.LastName)
	if err != nil {
		return ShooterImportResult{}, err
	}
	if in.UpdateShooterID != 0 {
		for _, m := range matches {
			if m.ID == in.UpdateShooterID {
				return s.importUpdate(ctx, m, sh)
			}
		}
		return ShooterImportResult{}, ErrInvalid
	}
	if len(matches) > 0 {
		return ShooterImportResult{}, &NameConflictError{Candidates: matches}
	}
	return s.importCreate(ctx, sh)
}

func (s *Service) importUpdate(ctx context.Context, existing, sh repository.Shooter) (ShooterImportResult, error) {
	if existing.Archived {
		return ShooterImportResult{}, &ArchivedShooterError{Shooter: existing}
	}
	id := existing.ID
	sh.ID = id
	if err := s.UpdateShooter(ctx, sh); err != nil {
		return ShooterImportResult{}, err
	}
	return ShooterImportResult{ID: id}, nil
}

func (s *Service) importCreate(ctx context.Context, sh repository.Shooter) (ShooterImportResult, error) {
	id, err := s.CreateShooter(ctx, sh)
	if err != nil {
		return ShooterImportResult{}, err
	}
	return ShooterImportResult{ID: id, Created: true}, nil
}
