package models

type Feed struct {
	URL   string
	Title string
}

type UserFeed struct {
	ID     int64
	UserID int64
	URL    string
	Title  string
}

type Post struct {
	Title     string
	URL       string
	FeedTitle string
	FeedURL   string
}

type UserSettings struct {
	UserID            int64
	AutoDigestHourUTC int64
}

type UserPosts struct {
	UserID int64
	Posts  []Post
}
