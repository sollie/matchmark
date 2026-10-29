package service

import (
	"context"
	"strings"
	"unicode"

	"github.com/sollie/matchmark/internal/report"
	"github.com/sollie/matchmark/internal/repository"
)

func (s *Service) loadReport(ctx context.Context, matchID int64) (report.Data, error) {
	comp, err := s.repo.GetMatch(ctx, matchID)
	if err != nil {
		return report.Data{}, mapRepoErr(err)
	}
	entries, err := s.repo.ReportEntries(ctx, matchID)
	if err != nil {
		return report.Data{}, err
	}
	for i := range entries {
		for j := range entries[i].Segments {
			if entries[i].Segments[j].Status != repository.StatusAccepted {
				entries[i].Segments[j].Counts = repository.RingCounts{}
			}
		}
	}
	return report.Data{Match: comp, Entries: entries}, nil
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

func (s *Service) renderWorkbook(ctx context.Context, matchID int64) (Download, error) {
	data, err := s.loadReport(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	out, err := report.Workbook(data)
	if err != nil {
		return Download{}, err
	}
	return Download{BaseName: downloadBaseName(data.Match.Name, data.Match.Date), Data: out}, nil
}

func (s *Service) renderCSV(ctx context.Context, matchID int64) (Download, error) {
	data, err := s.loadReport(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	out, err := report.CSV(data)
	if err != nil {
		return Download{}, err
	}
	return Download{BaseName: downloadBaseName(data.Match.Name, data.Match.Date), Data: out}, nil
}

func (s *Service) ResultsWorkbook(ctx context.Context, matchID int64) (Download, error) {
	file, err := s.renderWorkbook(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("match", matchID), Details: "xlsx"}})
	return file, nil
}

func (s *Service) ResultsCSV(ctx context.Context, matchID int64) (Download, error) {
	file, err := s.renderCSV(ctx, matchID)
	if err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("match", matchID), Details: "csv"}})
	return file, nil
}
