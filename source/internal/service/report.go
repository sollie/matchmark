package service

import (
	"context"
	"strings"
	"unicode"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/repository"
)

func (s *Service) loadReport(ctx context.Context, competitionID int64) (repository.Competition, []repository.ReportEntry, error) {
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return repository.Competition{}, nil, mapRepoErr(err)
	}
	entries, err := s.repo.ReportEntries(ctx, competitionID)
	if err != nil {
		return repository.Competition{}, nil, err
	}
	return comp, entries, nil
}

type Download struct {
	BaseName string
	Data     []byte
}

const maxNameRunes = 60

func downloadBaseName(name, date string) string {
	var b strings.Builder
	pendingDash := false
	count := 0
	for _, r := range name {
		if count >= maxNameRunes {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			if pendingDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingDash = false
			b.WriteRune(r)
			count++
		} else {
			pendingDash = true
		}
	}
	base := b.String()
	if base == "" {
		base = "results"
	}
	return base + "-" + date
}

func (s *Service) ResultsWorkbook(ctx context.Context, competitionID int64) (Download, error) {
	comp, entries, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	out, err := report.Workbook(report.Build(comp, entries))
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "xlsx"}})
	return Download{BaseName: downloadBaseName(comp.Name, comp.Date), Data: out}, nil
}

func (s *Service) ResultsCSV(ctx context.Context, competitionID int64) (Download, error) {
	comp, entries, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	out, err := report.CSV(report.Build(comp, entries))
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "csv"}})
	return Download{BaseName: downloadBaseName(comp.Name, comp.Date), Data: out}, nil
}
