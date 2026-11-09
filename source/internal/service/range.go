package service

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/sollie/matchmark/internal/repository"
)

const MaxRangeNameLength = 100

type Range = repository.Range

type OptionalID struct {
	Set bool
	ID  *int64
}

func (o *OptionalID) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.ID = nil
		return nil
	}
	var id int64
	if err := json.Unmarshal(data, &id); err != nil {
		return err
	}
	o.ID = &id
	return nil
}

func cleanRangeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxRangeNameLength {
		return "", ErrInvalid
	}
	return name, nil
}

func (s *Service) ListRanges(ctx context.Context) ([]Range, error) {
	rs, err := s.repo.ListRanges(ctx)
	return rs, mapRepoErr(err)
}

func (s *Service) GetRange(ctx context.Context, id int64) (Range, error) {
	g, err := s.repo.GetRange(ctx, id)
	return g, mapRepoErr(err)
}

func (s *Service) CreateRange(ctx context.Context, name string) (int64, error) {
	name, err := cleanRangeName(name)
	if err != nil {
		return 0, err
	}
	id, err := s.repo.CreateRange(ctx, name)
	if err != nil {
		return 0, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "range.create", Target: idTarget("range", id), Details: name}})
	return id, nil
}

func (s *Service) UpdateRange(ctx context.Context, id int64, name string) error {
	name, err := cleanRangeName(name)
	if err != nil {
		return err
	}
	if err := s.repo.RenameRange(ctx, id, name); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "range.update", Target: idTarget("range", id), Details: name}})
	return nil
}

func (s *Service) SetRangeArchived(ctx context.Context, id int64, archived bool) error {
	if _, err := s.repo.GetRange(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	action := "range.unarchive"
	if archived {
		_, isDefault, err := s.repo.RangeUsage(ctx, id)
		if err != nil {
			return mapRepoErr(err)
		}
		if isDefault {
			return ErrConflict
		}
		action = "range.archive"
	}
	if err := s.repo.SetRangeArchived(ctx, id, archived); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: action, Target: idTarget("range", id)}})
	return nil
}

func (s *Service) DeleteRange(ctx context.Context, id int64) error {
	if _, err := s.repo.GetRange(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	used, isDefault, err := s.repo.RangeUsage(ctx, id)
	if err != nil {
		return mapRepoErr(err)
	}
	if used > 0 || isDefault {
		return ErrConflict
	}
	if err := s.repo.DeleteRange(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "range.delete", Target: idTarget("range", id)}})
	return nil
}

func (s *Service) setDefaultRange(ctx context.Context, id *int64) error {
	if id != nil {
		g, err := s.repo.GetRange(ctx, *id)
		if err != nil || g.Archived {
			return ErrInvalid
		}
	}
	if err := s.repo.SetClubDefaultRange(ctx, id); err != nil {
		return mapRepoErr(err)
	}
	details := ""
	if id != nil {
		details = idTarget("range", *id)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "club.default_range", Target: "club", Details: details}})
	return nil
}

func (s *Service) newCompetitionRange(ctx context.Context, rng OptionalID) (*int64, error) {
	if rng.Set {
		return s.activeRange(ctx, rng.ID, nil)
	}
	club, err := s.repo.GetClub(ctx)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	if club.DefaultRangeID != nil {
		return club.DefaultRangeID, nil
	}
	all, err := s.repo.ListRanges(ctx)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	var only *int64
	for _, g := range all {
		if g.Archived {
			continue
		}
		if only != nil {
			return nil, nil
		}
		id := g.ID
		only = &id
	}
	return only, nil
}

func (s *Service) updatedCompetitionRange(ctx context.Context, rng OptionalID, current *int64) (*int64, error) {
	if !rng.Set {
		return current, nil
	}
	return s.activeRange(ctx, rng.ID, current)
}

func (s *Service) activeRange(ctx context.Context, id, current *int64) (*int64, error) {
	if id == nil {
		return nil, nil
	}
	g, err := s.repo.GetRange(ctx, *id)
	if err != nil {
		return nil, ErrInvalid
	}
	if g.Archived && (current == nil || *current != g.ID) {
		return nil, ErrInvalid
	}
	return id, nil
}
