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

func (s *Service) resolveOfficials(ctx context.Context, c *repository.Match, current repository.Match) error {
	var err error
	if c.RefereeWAID, c.RefereeName, err = s.resolveOfficial(ctx, c.RefereeWAID, current.RefereeWAID, current.RefereeName); err != nil {
		return err
	}
	c.RangeOfficerWAID, c.RangeOfficerName, err = s.resolveOfficial(ctx, c.RangeOfficerWAID, current.RangeOfficerWAID, current.RangeOfficerName)
	return err
}

func officialsDetails(oldMatch, newMatch repository.Match) string {
	var parts []string
	if newMatch.RefereeWAID != oldMatch.RefereeWAID {
		parts = append(parts, "referee "+officialLabel(newMatch.RefereeWAID))
	}
	if newMatch.RangeOfficerWAID != oldMatch.RangeOfficerWAID {
		parts = append(parts, "range officer "+officialLabel(newMatch.RangeOfficerWAID))
	}
	return strings.Join(parts, ", ")
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
