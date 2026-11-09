package service

import (
	"context"
	"time"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/rulebook"
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
		Range:       view.Match.Range,
		Segments:    view.Segments,
	}
	match, err := s.repo.GetMatch(ctx, view.Match.ID)
	if err != nil {
		return Download{}, mapRepoErr(err)
	}
	if data.Competition, err = s.repo.CompetitionName(ctx, match.CompetitionID); err != nil {
		return Download{}, mapRepoErr(err)
	}
	data.Referee = officialDisplay(match.RefereeName, match.RefereeWAID)
	data.RangeOfficer = officialDisplay(match.RangeOfficerName, match.RangeOfficerWAID)
	var logo ClubLogo
	if data.ClubName, logo, err = s.clubBranding(ctx); err != nil {
		return Download{}, err
	}
	data.Logo, data.LogoType = logo.Data, logo.ContentType
	if course, err := rulebook.CourseByKey(view.Match.Course); err == nil {
		byNumber := map[int]string{}
		for _, seg := range course.Segments {
			byNumber[seg.Number] = seg.Short
		}
		for _, n := range view.Segments {
			data.Headings = append(data.Headings, byNumber[n])
		}
		for _, g := range course.SumGroups {
			data.SumGroups = append(data.SumGroups, report.ResultsPDFSumGroup{Heading: g.Heading, Segments: g.Segments})
		}
	}
	for _, c := range view.Categories {
		sections := []ResultsGroup{{Class: -1, Rows: c.Rows}}
		if len(c.Groups) > 0 {
			sections = c.Groups
		}
		cat := report.ResultsPDFCategory{Category: c.Category}
		for i, g := range sections {
			group := report.ResultsPDFGroup{}
			if g.Class >= 0 {
				group.Class = g.Class.Name()
			}
			for _, r := range g.Rows {
				group.Rows = append(group.Rows, pdfRow(r))
			}
			if i == len(sections)-1 {
				for _, r := range c.DidNotFinish {
					group.Rows = append(group.Rows, pdfDNFRow(r))
				}
			}
			if len(group.Rows) > 0 {
				cat.Groups = append(cat.Groups, group)
			}
		}
		if len(cat.Groups) > 0 {
			data.Categories = append(data.Categories, cat)
		}
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

func pdfRow(r ResultsRow) report.ResultsPDFRow {
	row := report.ResultsPDFRow{
		Rank: r.Rank, LastName: r.LastName, FirstName: r.FirstName, WAID: r.WAID,
		Club: r.Club, Country: r.Country, Remarks: r.Remarks, Total: r.Total, XCount: r.XCount,
	}
	if r.Class != nil {
		name := r.Class.Name()
		row.Classification = &name
	}
	for _, st := range r.Segments {
		row.Segments = append(row.Segments, report.ResultsPDFSegment{Segment: st.Segment, Score: st.Score})
	}
	return row
}

func officialDisplay(name, waid string) string {
	if name != "" {
		return name
	}
	return waid
}

func pdfDNFRow(r ResultsDNFRow) report.ResultsPDFRow {
	row := report.ResultsPDFRow{
		LastName: r.LastName, FirstName: r.FirstName, WAID: r.WAID,
		Club: r.Club, Country: r.Country, Remarks: r.Remarks, DNF: true,
	}
	for _, st := range r.Segments {
		row.Segments = append(row.Segments, report.ResultsPDFSegment{Segment: st.Segment, Score: st.Score})
	}
	return row
}
