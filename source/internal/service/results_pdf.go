package service

import (
	"context"
	"time"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/repository"
)

func (s *Service) PublicResultsPDF(ctx context.Context, competitionID int64) (Download, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return Download{}, mapRepoErr(err)
	}
	if CompetitionStatus(comp.Status) != StatusFinished {
		return Download{}, ErrNotFound
	}
	vis, err := s.repo.GetVisibility(ctx, competitionID)
	if err != nil {
		return Download{}, mapRepoErr(err)
	}
	return s.renderResultsPDF(ctx, comp, vis)
}

func (s *Service) AdminResultsPDF(ctx context.Context, competitionID int64) (Download, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return Download{}, mapRepoErr(err)
	}
	file, err := s.renderResultsPDF(ctx, comp, showAll)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "pdf"}})
	return file, nil
}

func (s *Service) renderResultsPDF(ctx context.Context, comp repository.Competition, vis repository.Visibility) (Download, error) {
	res, err := s.resultsProjection(ctx, comp, vis)
	if err != nil {
		return Download{}, err
	}
	data := report.ResultsPDFData{Generated: time.Now(), Results: res}
	var logo ClubLogo
	if data.ClubName, logo, err = s.clubBranding(ctx); err != nil {
		return Download{}, err
	}
	data.Logo, data.LogoType = logo.Data, logo.ContentType
	pdf, err := report.ResultsPDF(data)
	if err != nil {
		return Download{}, err
	}
	return Download{BaseName: downloadBaseName(comp.Name, comp.Date) + "-results", Data: pdf}, nil
}
