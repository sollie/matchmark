package service

import (
	"context"
	"time"

	"github.com/sollie/matchmark/internal/report"
)

func (s *Service) PublicResultsPDF(ctx context.Context, competitionID int64) (Download, error) {
	view, err := s.PublicResults(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	return s.renderResultsPDF(ctx, view)
}

func (s *Service) AdminResultsPDF(ctx context.Context, competitionID int64) (Download, error) {
	view, err := s.AdminResults(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	file, err := s.renderResultsPDF(ctx, view)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "pdf"}})
	return file, nil
}

func (s *Service) renderResultsPDF(ctx context.Context, view ResultsView) (Download, error) {
	data := report.ResultsPDFData{
		Generated:   time.Now(),
		Name:        view.Competition.Name,
		Date:        view.Competition.Date,
		CourseLabel: view.Competition.CourseLabel,
		Stages:      view.Stages,
	}
	var logo ClubLogo
	var err error
	if data.ClubName, logo, err = s.clubBranding(ctx); err != nil {
		return Download{}, err
	}
	data.Logo, data.LogoType = logo.Data, logo.ContentType
	for _, c := range view.Categories {
		cat := report.ResultsPDFCategory{Category: c.Category}
		for _, r := range c.Rows {
			row := report.ResultsPDFRow{
				Rank: r.Rank, LastName: r.LastName, FirstName: r.FirstName, WAID: r.WAID,
				Club: r.Club, Country: r.Country, Total: r.Total, XCount: r.XCount,
			}
			for _, st := range r.Stages {
				row.Stages = append(row.Stages, report.ResultsPDFStage{Stage: st.Stage, Score: st.Score})
			}
			cat.Rows = append(cat.Rows, row)
		}
		data.Categories = append(data.Categories, cat)
	}
	for _, n := range view.NewShooters {
		data.NewShooters = append(data.NewShooters, report.ResultsPDFNewShooter{
			LastName: n.LastName, FirstName: n.FirstName, Category: n.Category,
			Club: n.Club, Country: n.Country, Total: n.Total, XCount: n.XCount,
		})
	}
	pdf, err := report.ResultsPDF(data)
	if err != nil {
		return Download{}, err
	}
	return Download{BaseName: downloadBaseName(view.Competition.Name, view.Competition.Date) + "-results", Data: pdf}, nil
}
