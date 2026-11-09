package service

import (
	"context"
	"errors"
	"strings"

	"github.com/sollie/matchmark/internal/repository"
	"github.com/sollie/matchmark/internal/wa1500"
)

const maxWAIDLength = 32

var ErrWAIDNotFound = errors.New("WAID not found in the WA1500 register")

func (s *Service) resolveOfficial(ctx context.Context, waid, currentWAID, currentName string) (string, string, error) {
	waid = strings.TrimSpace(waid)
	if waid == "" {
		return "", "", nil
	}
	if len(waid) > maxWAIDLength {
		return "", "", ErrInvalid
	}
	if waid == currentWAID && currentName != "" {
		return waid, currentName, nil
	}
	if s.register == nil {
		return "", "", wa1500.ErrUpstream
	}
	rows, err := s.register.Search(ctx, waid)
	if err != nil {
		return "", "", errors.Join(wa1500.ErrUpstream, err)
	}
	for _, row := range rows {
		if row.WAID != waid {
			continue
		}
		name := strings.TrimSpace(NormalizeFirstName(row.FirstName) + " " + NormalizeLastName(row.LastName))
		if name == "" {
			break
		}
		return waid, name, nil
	}
	return "", "", ErrWAIDNotFound
}

func (s *Service) resolveOfficialList(ctx context.Context, requested, current []repository.Official) ([]repository.Official, error) {
	known := make(map[string]string, len(current))
	for _, o := range current {
		known[o.WAID] = o.Name
	}
	out := []repository.Official{}
	seen := map[string]bool{}
	for _, o := range requested {
		waid := strings.TrimSpace(o.WAID)
		if waid == "" {
			continue
		}
		if seen[waid] {
			return nil, ErrInvalid
		}
		seen[waid] = true
		resolved, name, err := s.resolveOfficial(ctx, waid, waid, known[waid])
		if err != nil {
			return nil, err
		}
		out = append(out, repository.Official{WAID: resolved, Name: name})
	}
	return out, nil
}

func (s *Service) resolveOfficials(ctx context.Context, c *repository.Match, current repository.Match) error {
	var err error
	if c.Referees, err = s.resolveOfficialList(ctx, c.Referees, current.Referees); err != nil {
		return err
	}
	c.RangeOfficers, err = s.resolveOfficialList(ctx, c.RangeOfficers, current.RangeOfficers)
	return err
}

func officialsDetails(oldMatch, newMatch repository.Match) string {
	var parts []string
	if !sameOfficials(oldMatch.Referees, newMatch.Referees) {
		parts = append(parts, "referees "+officialsLabel(newMatch.Referees))
	}
	if !sameOfficials(oldMatch.RangeOfficers, newMatch.RangeOfficers) {
		parts = append(parts, "range officers "+officialsLabel(newMatch.RangeOfficers))
	}
	return strings.Join(parts, ", ")
}

func sameOfficials(a, b []repository.Official) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].WAID != b[i].WAID {
			return false
		}
	}
	return true
}

func officialsLabel(list []repository.Official) string {
	if len(list) == 0 {
		return "cleared"
	}
	waids := make([]string, len(list))
	for i, o := range list {
		waids[i] = o.WAID
	}
	return strings.Join(waids, ", ")
}

func officialLabel(waid string) string {
	if waid == "" {
		return "cleared"
	}
	return waid
}

func joinDetails(base, extra string) string {
	if extra == "" {
		return base
	}
	return base + ", " + extra
}
