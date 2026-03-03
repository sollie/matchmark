package db

import (
	"database/sql"
	"fmt"

	"github.com/sollie/matchmark/internal/auth"
)

func Seed(db *sql.DB, adminPassword string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count); err != nil {
		return fmt.Errorf("check admin existence: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	if _, err := db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')`,
		"admin", hash,
	); err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	return nil
}
