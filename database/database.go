package database

import (
	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"

	"log/slog"
	model "telekilogram/model"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) AddFeed(userID int64, feedURL string) error {
	query := "INSERT OR IGNORE INTO feeds (user_id, url) VALUES (?, ?)"
	_, err := d.db.Exec(query, userID, feedURL)
	return err
}

func (d *Database) RemoveFeed(feedID int64) error {
	query := "DELETE FROM feeds WHERE id = ?"
	_, err := d.db.Exec(query, feedID)
	return err
}

func (d *Database) GetUserFeeds(userID int64) ([]model.Feed, error) {
	query := "SELECT id, url FROM feeds WHERE user_id = ?"
	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", slog.Any("error", err))
		}
	}()

	var feeds []model.Feed
	for rows.Next() {
		var f model.Feed
		if err := rows.Scan(&f.ID, &f.URL); err != nil {
			return nil, err
		}

		f.UserID = userID
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (d *Database) GetAllFeeds() ([]model.Feed, error) {
	query := `SELECT id, user_id, url FROM feeds`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", slog.Any("error", err))
		}
	}()

	var feeds []model.Feed
	for rows.Next() {
		var f model.Feed
		if err := rows.Scan(&f.ID, &f.UserID, &f.URL); err != nil {
			return nil, err
		}

		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
