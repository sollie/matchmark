package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/sollie/matchmark/internal/export"
	"github.com/sollie/matchmark/internal/repository"
)

type ExportService struct {
	exportRepo   *repository.ExportRepository
	competitions *repository.CompetitionRepository
}

func NewExportService(exportRepo *repository.ExportRepository, competitions *repository.CompetitionRepository) *ExportService {
	return &ExportService{exportRepo: exportRepo, competitions: competitions}
}

func (s *ExportService) ExportJSON(ctx context.Context, competitionID int64) ([]byte, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}
	shooters, err := s.exportRepo.GetCompetitionExport(ctx, competitionID)
	if err != nil {
		return nil, fmt.Errorf("export data: %w", err)
	}
	var buf bytes.Buffer
	if err := export.WriteJSON(&buf, shooters); err != nil {
		return nil, fmt.Errorf("write json: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *ExportService) ExportCSV(ctx context.Context, competitionID int64) ([]byte, error) {
	if _, err := s.competitions.GetByID(ctx, competitionID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}
	shooters, err := s.exportRepo.GetCompetitionExport(ctx, competitionID)
	if err != nil {
		return nil, fmt.Errorf("export data: %w", err)
	}
	var buf bytes.Buffer
	if err := export.WriteCSV(&buf, shooters); err != nil {
		return nil, fmt.Errorf("write csv: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *ExportService) ExportPDF(ctx context.Context, competitionID int64) ([]byte, error) {
	comp, err := s.competitions.GetByID(ctx, competitionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrCompetitionNotFound
		}
		return nil, fmt.Errorf("get competition: %w", err)
	}
	shooters, err := s.exportRepo.GetCompetitionExport(ctx, competitionID)
	if err != nil {
		return nil, fmt.Errorf("export data: %w", err)
	}
	var buf bytes.Buffer
	if err := export.WritePDF(&buf, comp.Name, comp.Date, shooters); err != nil {
		return nil, fmt.Errorf("write pdf: %w", err)
	}
	return buf.Bytes(), nil
}
