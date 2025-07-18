package database

import (
	"database/sql"
	"embed"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	"log/slog"
	"telekilogram/model"
)

type Database struct {
	db *sql.DB
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return nil, err
	}

	srcDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		srcDriver,
		"sqlite3",
		dbDriver,
	)
	if err != nil {
		return nil, err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}

	return &Database{db: db}, nil
}

func (d *Database) AddFeed(userID int64, feedURL string, feedTitle string) error {
	query := "insert or ignore into feeds (user_id, url, title) values (?, ?, ?)"
	_, err := d.db.Exec(query, userID, feedURL, feedTitle)
	return err
}

func (d *Database) UpdateFeedTitle(feedID int64, feedTitle string) error {
	query := "update feeds set title = ? where id = ?"
	_, err := d.db.Exec(query, feedTitle, feedID)
	return err
}

func (d *Database) RemoveFeed(feedID int64) error {
	query := "delete from feeds where id = ?"
	_, err := d.db.Exec(query, feedID)
	return err
}

func (d *Database) GetUserFeeds(userID int64) ([]model.UserFeed, error) {
	query := "select id, url, title from feeds where user_id = ?"
	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", slog.Any("error", err))
		}
	}()

	var feeds []model.UserFeed
	for rows.Next() {
		var f model.UserFeed
		if err := rows.Scan(&f.ID, &f.URL, &f.Title); err != nil {
			return nil, err
		}

		f.UserID = userID
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (d *Database) GetAllFeeds() ([]model.UserFeed, error) {
	query := `select id, user_id, url, title from feeds`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", slog.Any("error", err))
		}
	}()

	var feeds []model.UserFeed
	for rows.Next() {
		var f model.UserFeed
		if err := rows.Scan(&f.ID, &f.UserID, &f.URL, &f.Title); err != nil {
			return nil, err
		}

		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}
