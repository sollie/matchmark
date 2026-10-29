package service

import (
	"context"
	"time"

	"github.com/sollie/matchmark/internal/report"
)

func (s *Service) PublicResultsPDF(ctx context.Context, matchID int64) (Download, error) {
	view, err := s.PublicResults(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	return s.renderResultsPDF(ctx, view)
}

func (s *Service) adminResultsPDF(ctx context.Context, matchID int64) (Download, error) {
	view, err := s.AdminResults(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	return s.renderResultsPDF(ctx, view)
}

func (s *Service) AdminResultsPDF(ctx context.Context, matchID int64) (Download, error) {
	file, err := s.adminResultsPDF(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("match", matchID), Details: "pdf"}})
	return file, nil
}

func (s *Service) renderResultsPDF(ctx context.Context, view ResultsView) (Download, error) {
	data := report.ResultsPDFData{
		Generated:   time.Now(),
		Name:        view.Match.Name,
		Date:        view.Match.Date,
		CourseLabel: view.Match.CourseLabel,
		Segments:    view.Segments,
	}
	var logo ClubLogo
	var err error
	if data.ClubName, logo, err = s.clubBranding(ctx); err != nil {
		return Download{}, err
	}
	data.Logo, data.LogoType = logo.Data, logo.ContentType
	names := segmentNames(view.Match.Course)
	for _, n := range view.Segments {
		data.SegmentNames = append(data.SegmentNames, names[n])
	}
	for _, c := range view.Categories {
		cat := report.ResultsPDFCategory{Category: c.Category}
		for _, r := range c.Rows {
			row := report.ResultsPDFRow{
				Rank: r.Rank, LastName: r.LastName, FirstName: r.FirstName, WAID: r.WAID,
				Club: r.Club, Country: r.Country, Total: r.Total, XCount: r.XCount,
			}
			for _, st := range r.Segments {
				seg := report.ResultsPDFSegment{Segment: st.Segment, Score: st.Score}
				if c := st.RingCounts; c != nil {
					seg.Counts = []int{c.X, c.Ten, c.Nine, c.Eight, c.Seven, c.Zero, c.Miss}
				}
				row.Segments = append(row.Segments, seg)
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
	return Download{BaseName: downloadBaseName(view.Match.Name, view.Match.Date) + "-results", Data: pdf}, nil
}
