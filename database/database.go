package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	"log/slog"
)

type Database struct {
	db *sql.DB
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func New(dbPath string) (*Database, error) {
	dbFile, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB file: %w", err)
	}

	dbInstance, err := sqlite3.WithInstance(dbFile, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create DB instance: %w", err)
	}

	srcInstance, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create source instance: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		srcInstance,
		"sqlite3",
		dbInstance,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return nil, fmt.Errorf("failed to apply migrations: %w", err)
		}
		slog.Info("No migrations to apply")
	}

	slog.Info("DB is migrated")

	return &Database{db: dbFile}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
