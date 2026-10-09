package domain

import "time"

// ContentFeedback is a standalone message submitted by a user for future review.
type ContentFeedback struct {
	ID        int64
	FullName  string
	IP        string
	Message   string
	CreatedAt time.Time
}
