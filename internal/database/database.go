package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	dbsql "telekilogram/internal/database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Required by the library implementation.
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3" // Required by the library implementation.
)

type Database struct {
	q   *dbsql.Queries
	log *slog.Logger
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func New(ctx context.Context, dbPath string, log *slog.Logger) (*Database, error) {
	dbFile, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open DB file: %w", err)
	}

	dbInstance, err := sqlite3.WithInstance(dbFile, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("create DB instance: %w", err)
	}

	srcInstance, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create source instance: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", srcInstance, "sqlite3", dbInstance)
	if err != nil {
		return nil, fmt.Errorf("create migrate instance: %w", err)
	}

	migrateErr := m.Up()

	version, dirty, versionErr := m.Version()
	fields := []any{
		"dbPath", dbPath,
	}

	if versionErr == nil {
		fields = append(fields, "version", version, "dirty", dirty)
	} else if !errors.Is(versionErr, migrate.ErrNilVersion) {
		log.WarnContext(ctx, "Failed to fetch migration version",
			"error", versionErr,
			"dbPath", dbPath)
	}

	if migrateErr != nil {
		if !errors.Is(migrateErr, migrate.ErrNoChange) {
			return nil, fmt.Errorf("apply migrations: %w", migrateErr)
		}

		log.InfoContext(ctx, "No migrations to apply", fields...)
	} else {
		log.InfoContext(ctx, "DB is migrated", fields...)
	}

	q := dbsql.New(dbFile)
	return &Database{q: q, log: log}, nil
}
