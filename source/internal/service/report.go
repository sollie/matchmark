package service

import (
	"context"

	"github.com/sollie/matchmark/internal/report"
)

func (s *Service) loadReport(ctx context.Context, competitionID int64) (report.Data, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return report.Data{}, mapRepoErr(err)
	}
	entries, err := s.repo.ReportEntries(ctx, competitionID)
	if err != nil {
		return report.Data{}, err
	}
	return report.Data{Competition: comp, Entries: entries}, nil
}

func (s *Service) ResultsWorkbook(ctx context.Context, competitionID int64) ([]byte, error) {
	data, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	return report.Workbook(data)
}

func (s *Service) ResultsCSV(ctx context.Context, competitionID int64) ([]byte, error) {
	data, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	return report.CSV(data)
}
