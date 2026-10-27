package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	turso "turso.tech/database/tursogo"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Open(path string) (*sql.DB, error) {
	probe, err := sql.Open("turso", path+"?_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	conn := sql.OpenDB(&connector{drv: probe.Driver(), dsn: path + "?_busy_timeout=5000"})
	_ = probe.Close()
	return finishOpen(conn)
}

func OpenReplicated(ctx context.Context, path, remoteURL string) (*sql.DB, *turso.TursoSyncDb, error) {
	bootstrap := false
	sdb, err := turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
		Path:             path + "?_busy_timeout=5000",
		RemoteUrl:        remoteURL,
		BootstrapIfEmpty: &bootstrap,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("open replicated database: %w", err)
	}
	conn, err := sdb.Connect(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("open replicated database: %w", err)
	}
	conn.SetMaxIdleConns(1)
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	conn, err = finishOpen(conn)
	if err != nil {
		return nil, nil, err
	}
	return conn, sdb, nil
}

func finishOpen(conn *sql.DB) (*sql.DB, error) {
	conn.SetMaxOpenConns(1)
	if err := migrate(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

type connector struct {
	drv driver.Driver
	dsn string
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	dc, err := c.drv.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	stmt, err := dc.Prepare(`PRAGMA foreign_keys = ON`)
	if err != nil {
		_ = dc.Close()
		return nil, err
	}
	defer func() { _ = stmt.Close() }()
	se, ok := stmt.(driver.StmtExecContext)
	if !ok {
		_ = dc.Close()
		return nil, errors.New("driver statement does not support ExecContext")
	}
	if _, err := se.ExecContext(ctx, nil); err != nil {
		_ = dc.Close()
		return nil, err
	}
	return dc, nil
}

func (c *connector) Driver() driver.Driver { return c.drv }

func migrate(conn *sql.DB) error {
	if _, err := conn.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return fmt.Errorf("migration %s: invalid version prefix", name)
		}
		var applied int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied > 0 {
			continue
		}
		body, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := apply(conn, version, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func apply(conn *sql.DB, version int, body string) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(body); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
		return err
	}
	return tx.Commit()
}
