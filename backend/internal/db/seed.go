package db

import (
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"

	"crypto/rand"
	"encoding/base64"
)

const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	saltLen       = 16
)

func Seed(db *sql.DB, adminPassword string) error {
	if len(adminPassword) < 16 {
		return errors.New("ADMIN_PASSWORD must be at least 16 characters")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&count); err != nil {
		return fmt.Errorf("check admin existence: %w", err)
	}
	if count > 0 {
		return nil
	}

	hash, err := hashPassword(adminPassword)
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

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)

	return encoded, nil
}
