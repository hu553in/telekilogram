package database

import (
	"fmt"
	"log/slog"
	"telekilogram/models"
)

func (d *Database) AddFeed(
	userID int64,
	feedURL string,
	feedTitle string,
) error {
	query := "insert or ignore into feeds (user_id, url, title) values (?, ?, ?)"
	if _, err := d.db.Exec(query, userID, feedURL, feedTitle); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (d *Database) UpdateFeedTitle(feedID int64, feedTitle string) error {
	query := "update feeds set title = ? where id = ?"
	if _, err := d.db.Exec(query, feedTitle, feedID); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (d *Database) RemoveFeed(feedID int64) error {
	query := "delete from feeds where id = ?"
	if _, err := d.db.Exec(query, feedID); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (d *Database) GetUserFeeds(userID int64) ([]models.UserFeed, error) {
	query := "select id, url, title from feeds where user_id = ?"
	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows",
				slog.Any("err", err),
				slog.Int64("userID", userID))
		}
	}()

	var feeds []models.UserFeed
	for rows.Next() {
		var f models.UserFeed
		if err := rows.Scan(&f.ID, &f.URL, &f.Title); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		f.UserID = userID
		feeds = append(feeds, f)
	}

	return feeds, nil
}

func (d *Database) GetHourFeeds(hourUTC int64) ([]models.UserFeed, error) {
	var query string

	if hourUTC == 0 {
		query = `select f.id, f.user_id, f.url, f.title
		from feeds as f
		left join user_settings as us
		on us.user_id = f.user_id
		where us.user_id is null
		or us.auto_digest_hour_utc = ?`
	} else {
		query = `select f.id, f.user_id, f.url, f.title
		from feeds as f
		left join user_settings as us
		on us.user_id = f.user_id
		where us.auto_digest_hour_utc = ?`
	}

	rows, err := d.db.Query(query, hourUTC)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows",
				slog.Any("err", err),
				slog.Int64("hourUTC", hourUTC))
		}
	}()

	var feeds []models.UserFeed
	for rows.Next() {
		var f models.UserFeed
		if err := rows.Scan(&f.ID, &f.UserID, &f.URL, &f.Title); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		feeds = append(feeds, f)
	}

	return feeds, nil
}

func (d *Database) GetUserSettingsWithDefault(
	userID int64,
) (*models.UserSettings, error) {
	query := `select user_id, auto_digest_hour_utc
	from user_settings
	where user_id = ?`

	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows",
				slog.Any("err", err),
				slog.Int64("userID", userID))
		}
	}()

	if !rows.Next() {
		return &models.UserSettings{
			UserID:            userID,
			AutoDigestHourUTC: 0,
		}, nil
	}

	var us models.UserSettings
	if err := rows.Scan(&us.UserID, &us.AutoDigestHourUTC); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	return &us, nil
}

func (d *Database) UpsertUserSettings(userSettings *models.UserSettings) error {
	query := `insert into user_settings (user_id, auto_digest_hour_utc)
	values (?, ?)
	on conflict (user_id) do update
	set auto_digest_hour_utc = excluded.auto_digest_hour_utc`

	if _, err := d.db.Exec(
		query,
		userSettings.UserID,
		userSettings.AutoDigestHourUTC,
	); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}
