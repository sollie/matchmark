package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/model"
	"github.com/sollie/matchmark/internal/repository"
)

var (
	ErrCompetitionNotFound     = errors.New("competition not found")
	ErrInvalidStatus           = errors.New("invalid competition status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

var validStatuses = map[string]bool{
	"draft":    true,
	"active":   true,
	"finished": true,
}

var allowedTransitions = map[string]string{
	"draft":  "active",
	"active": "finished",
}

type CompetitionService struct {
	competitions *repository.CompetitionRepository
}

func NewCompetitionService(competitions *repository.CompetitionRepository) *CompetitionService {
	return &CompetitionService{competitions: competitions}
}

func (s *CompetitionService) GetByID(ctx context.Context, id int64) (*model.Competition, error) {
	c, err := s.competitions.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}
	return c, nil
}

func (s *CompetitionService) List(ctx context.Context, status string) ([]model.Competition, error) {
	if status != "" && !validStatuses[status] {
		return nil, ErrInvalidStatus
	}
	return s.competitions.List(ctx, status)
}

type CreateCompetitionParams struct {
	Name             string
	Date             string
	LanesTotal       int
	SpectatorEnabled bool
}

func (s *CompetitionService) Create(ctx context.Context, p CreateCompetitionParams) (*model.Competition, error) {
	if p.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if p.Date == "" {
		return nil, fmt.Errorf("date is required")
	}
	if p.LanesTotal < 1 {
		return nil, fmt.Errorf("lanes_total must be at least 1")
	}
	return s.competitions.Create(ctx, p.Name, p.Date, p.LanesTotal, p.SpectatorEnabled)
}

type UpdateCompetitionParams struct {
	Name             *string
	Date             *string
	LanesTotal       *int
	Status           *string
	SpectatorEnabled *bool
}

func (s *CompetitionService) Update(ctx context.Context, id int64, p UpdateCompetitionParams) (*model.Competition, error) {
	current, err := s.competitions.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}

	name := current.Name
	date := current.Date
	lanesTotal := current.LanesTotal
	status := current.Status
	spectator := current.SpectatorEnabled

	if p.Name != nil {
		name = *p.Name
	}
	if p.Date != nil {
		date = *p.Date
	}
	if p.LanesTotal != nil {
		if *p.LanesTotal < 1 {
			return nil, fmt.Errorf("lanes_total must be at least 1")
		}
		lanesTotal = *p.LanesTotal
	}
	if p.Status != nil {
		if !validStatuses[*p.Status] {
			return nil, ErrInvalidStatus
		}
		allowed, ok := allowedTransitions[current.Status]
		if current.Status != *p.Status && (!ok || allowed != *p.Status) {
			return nil, ErrInvalidStatusTransition
		}
		status = *p.Status
	}
	if p.SpectatorEnabled != nil {
		spectator = *p.SpectatorEnabled
	}

	return s.competitions.Update(ctx, id, name, date, status, lanesTotal, spectator)
}
