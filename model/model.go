package model

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
