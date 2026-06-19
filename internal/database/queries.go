package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	dbsql "telekilogram/internal/database/sql"
	"telekilogram/internal/domain"
)

func (d *Database) AddFeed(
	ctx context.Context,
	userID int64,
	feedURL string,
	feedTitle string,
) error {
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return errors.New("feed URL is empty")
	}

	feedTitle = strings.TrimSpace(feedTitle)
	if feedTitle == "" {
		feedTitle = feedURL
	}

	err := d.q.AddOrIgnoreFeed(ctx, dbsql.AddOrIgnoreFeedParams{
		UserID: userID,
		Url:    feedURL,
		Title:  feedTitle,
	})
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	return nil
}

func (d *Database) UpdateFeedTitle(ctx context.Context, feedID int64, feedTitle string) error {
	feedTitle = strings.TrimSpace(feedTitle)
	if feedTitle == "" {
		return errors.New("feed title is empty")
	}

	err := d.q.UpdateFeedTitle(ctx, dbsql.UpdateFeedTitleParams{
		Title: feedTitle,
		ID:    feedID,
	})
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	return nil
}

func (d *Database) RemoveFeed(ctx context.Context, feedID int64) error {
	err := d.q.RemoveFeed(ctx, feedID)
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	return nil
}

func (d *Database) GetUserFeeds(ctx context.Context, userID int64) ([]domain.UserFeed, error) {
	rows, err := d.q.GetUserFeeds(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}

	var feeds []domain.UserFeed
	for _, r := range rows {
		var f domain.UserFeed

		f.ID = r.ID
		f.URL = strings.TrimSpace(r.Url)
		f.Title = strings.TrimSpace(r.Title)
		f.UserID = userID

		feeds = append(feeds, f)
	}

	return feeds, nil
}

func (d *Database) GetHourFeeds(ctx context.Context, hourUTC int64) ([]domain.UserFeed, error) {
	var rows []dbsql.Feed
	var err error

	if hourUTC == 0 {
		rows, err = d.q.GetHourFeedsMidnightUTC(ctx, hourUTC)
	} else {
		rows, err = d.q.GetHourFeeds(ctx, hourUTC)
	}
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}

	var feeds []domain.UserFeed
	for _, r := range rows {
		var f domain.UserFeed

		f.ID = r.ID
		f.URL = strings.TrimSpace(r.Url)
		f.Title = strings.TrimSpace(r.Title)
		f.UserID = r.UserID

		feeds = append(feeds, f)
	}

	return feeds, nil
}

func (d *Database) GetUserSettingsWithDefault(
	ctx context.Context,
	userID int64,
) (*domain.UserSettings, error) {
	row, err := d.q.GetUserSettings(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &domain.UserSettings{
				UserID:            userID,
				AutoDigestHourUTC: 0,
			}, nil
		}
		return nil, fmt.Errorf("execute query: %w", err)
	}

	return &domain.UserSettings{
		UserID:            row.UserID,
		AutoDigestHourUTC: row.AutoDigestHourUtc,
	}, nil
}

func (d *Database) UpsertUserSettings(ctx context.Context, userSettings *domain.UserSettings) error {
	err := d.q.UpsertUserSettings(ctx, dbsql.UpsertUserSettingsParams{
		UserID:            userSettings.UserID,
		AutoDigestHourUtc: userSettings.AutoDigestHourUTC,
	})
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
	}

	return nil
}
