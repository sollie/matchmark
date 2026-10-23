package service

import (
	"context"
	"strings"
	"unicode"

	"github.com/sollie/matchmark/internal/repository"
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
	for i := range entries {
		for j := range entries[i].Stages {
			if entries[i].Stages[j].Status != repository.StatusAccepted {
				entries[i].Stages[j].Counts = repository.RingCounts{}
			}
		}
	}
	return report.Data{Competition: comp, Entries: entries}, nil
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
	data, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	out, err := report.Workbook(data)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "xlsx"}})
	return Download{BaseName: downloadBaseName(data.Competition.Name, data.Competition.Date), Data: out}, nil
}

func (s *Service) ResultsCSV(ctx context.Context, competitionID int64) (Download, error) {
	data, err := s.loadReport(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}
	out, err := report.CSV(data)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "csv"}})
	return Download{BaseName: downloadBaseName(data.Competition.Name, data.Competition.Date), Data: out}, nil
}
