package repository

import "context"

type ClubSettings struct {
	Name    string
	HasLogo bool
	Version int64
}

type ClubLogo struct {
	Data        []byte
	ContentType string
}

func (r *Repository) GetClub(ctx context.Context) (ClubSettings, error) {
	var c ClubSettings
	err := r.db.QueryRowContext(ctx,
		`SELECT name, logo IS NOT NULL, version FROM club_settings WHERE id = 1`).Scan(&c.Name, &c.HasLogo, &c.Version)
	return c, mapErr(err)
}

func (r *Repository) SetClubName(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE club_settings SET name = ?, version = version + 1 WHERE id = 1`, name)
	return err
}

func (r *Repository) GetClubLogo(ctx context.Context) (ClubLogo, error) {
	var l ClubLogo
	err := r.db.QueryRowContext(ctx,
		`SELECT logo, logo_content_type FROM club_settings WHERE id = 1 AND logo IS NOT NULL`).Scan(&l.Data, &l.ContentType)
	return l, mapErr(err)
}

func (r *Repository) SetClubLogo(ctx context.Context, logo ClubLogo) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE club_settings SET logo = ?, logo_content_type = ?, version = version + 1 WHERE id = 1`,
		logo.Data, logo.ContentType)
	return err
}

func (r *Repository) DeleteClubLogo(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE club_settings SET logo = NULL, logo_content_type = '', version = version + 1 WHERE id = 1`)
	return err
}
