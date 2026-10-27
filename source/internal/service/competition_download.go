package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	FormatXLSX = "xlsx"
	FormatCSV  = "csv"
	FormatPDF  = "pdf"
	FormatAll  = "all"

	skippedFile = "SKIPPED.txt"
)

type matchRenderer func(ctx context.Context, matchID int64) (Download, error)

func (s *Service) renderers(format string) ([]string, map[string]matchRenderer, error) {
	all := map[string]matchRenderer{
		FormatXLSX: s.renderWorkbook,
		FormatCSV:  s.renderCSV,
		FormatPDF:  s.adminResultsPDF,
	}
	switch format {
	case FormatXLSX, FormatCSV, FormatPDF:
		return []string{format}, all, nil
	case FormatAll:
		return []string{FormatXLSX, FormatCSV, FormatPDF}, all, nil
	}
	return nil, nil, ErrInvalid
}

func (s *Service) CompetitionDownload(ctx context.Context, competitionID int64, format string) (Download, error) {
	exts, render, err := s.renderers(format)
	if err != nil {
		return Download{}, err
	}
	comp, err := s.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return Download{}, mapRepoErr(err)
	}
	matches, err := s.repo.ListCompetitionMatches(ctx, competitionID)
	if err != nil {
		return Download{}, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	used := map[string]bool{}
	var skipped []string
	for _, m := range matches {
		if MatchStatus(m.Status) != StatusFinished {
			skipped = append(skipped, fmt.Sprintf("%s (%s, course %s): %s", m.Name, m.Date, m.Course, m.Status))
			continue
		}
		base := uniqueName(used, downloadBaseName(m.Name, m.Date)+"-"+m.Course, m.ID)
		for _, ext := range exts {
			file, err := render[ext](ctx, m.ID)
			if err != nil {
				return Download{}, err
			}
			path := base + "." + ext
			if format == FormatAll {
				path = base + "/" + path
			}
			if err := addZipFile(zw, path, file.Data); err != nil {
				return Download{}, err
			}
		}
	}
	if len(skipped) > 0 {
		list := "Matches skipped because they are not finished:\n" + strings.Join(skipped, "\n") + "\n"
		if err := addZipFile(zw, skippedFile, []byte(list)); err != nil {
			return Download{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return Download{}, err
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "report.download", Target: idTarget("competition", competitionID), Details: "zip " + format}})
	return Download{BaseName: downloadBaseName(comp.Name, comp.StartDate) + "-" + format, Data: buf.Bytes()}, nil
}

func uniqueName(used map[string]bool, base string, id int64) string {
	if used[base] {
		base += "-" + strconv.FormatInt(id, 10)
	}
	used[base] = true
	return base
}

func addZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
