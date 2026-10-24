package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"unicode/utf8"

	"github.com/sollie/matchmark/internal/repository"
)

const (
	MaxClubNameLength = 100
	MaxLogoBytes      = 512 * 1024
	MaxLogoDimension  = 2048
)

type ClubSettings = repository.ClubSettings

type ClubLogo = repository.ClubLogo

func (s *Service) GetClub(ctx context.Context) (ClubSettings, error) {
	c, err := s.repo.GetClub(ctx)
	return c, mapRepoErr(err)
}

func (s *Service) clubBranding(ctx context.Context) (string, ClubLogo, error) {
	club, err := s.GetClub(ctx)
	if err != nil {
		return "", ClubLogo{}, err
	}
	if !club.HasLogo {
		return club.Name, ClubLogo{}, nil
	}
	logo, err := s.GetClubLogo(ctx)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", ClubLogo{}, err
	}
	return club.Name, logo, nil
}

func (s *Service) SetClubName(ctx context.Context, name string) (ClubSettings, error) {
	name = strings.TrimSpace(name)
	if utf8.RuneCountInString(name) > MaxClubNameLength {
		return ClubSettings{}, ErrInvalid
	}
	if err := s.repo.SetClubName(ctx, name); err != nil {
		return ClubSettings{}, mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "club.update", Target: "club", Details: name}})
	return s.GetClub(ctx)
}

func (s *Service) GetClubLogo(ctx context.Context) (ClubLogo, error) {
	l, err := s.repo.GetClubLogo(ctx)
	return l, mapRepoErr(err)
}

func (s *Service) SetClubLogo(ctx context.Context, contentType string, data []byte) error {
	if len(data) == 0 || len(data) > MaxLogoBytes {
		return ErrInvalid
	}
	if err := validateLogo(contentType, data); err != nil {
		return err
	}
	if err := s.repo.SetClubLogo(ctx, ClubLogo{Data: data, ContentType: contentType}); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "club.logo.set", Target: "club", Details: contentType}})
	return nil
}

func (s *Service) DeleteClubLogo(ctx context.Context) error {
	if err := s.repo.DeleteClubLogo(ctx); err != nil {
		return mapRepoErr(err)
	}
	s.changes.Record(ctx, Change{Audit: AuditEvent{Action: "club.logo.delete", Target: "club"}})
	return nil
}

func validateLogo(contentType string, data []byte) error {
	var decodeConfig func() (image.Config, error)
	switch contentType {
	case "image/png":
		decodeConfig = func() (image.Config, error) { return png.DecodeConfig(bytes.NewReader(data)) }
	case "image/jpeg":
		decodeConfig = func() (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(data)) }
	default:
		return ErrInvalid
	}
	cfg, err := decodeConfig()
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxLogoDimension || cfg.Height > MaxLogoDimension {
		return ErrInvalid
	}
	return nil
}
