package model

type Feed struct {
	ID     int64
	UserID int64
	URL    string
}

type Post struct {
	Title     string
	URL       string
	FeedTitle string
}
