package domain

import "time"

// ContentFeedbackType distinguishes reports about existing content from requests for new content.
type ContentFeedbackType string

const (
	ContentFeedbackTypeContentReport  ContentFeedbackType = "CONTENT_REPORT"
	ContentFeedbackTypeContentRequest ContentFeedbackType = "CONTENT_REQUEST"
)

type ContentFeedbackCategory string

const (
	ContentFeedbackCategoryIncorrect  ContentFeedbackCategory = "INCORRECT"
	ContentFeedbackCategoryIncomplete ContentFeedbackCategory = "INCOMPLETE"
	ContentFeedbackCategoryMissing    ContentFeedbackCategory = "MISSING"
	ContentFeedbackCategoryUnclear    ContentFeedbackCategory = "UNCLEAR"
	ContentFeedbackCategoryOutdated   ContentFeedbackCategory = "OUTDATED"
	ContentFeedbackCategoryOther      ContentFeedbackCategory = "OTHER"
)

type ContentFeedbackStatus string

const (
	ContentFeedbackStatusPending  ContentFeedbackStatus = "PENDING"
	ContentFeedbackStatusInReview ContentFeedbackStatus = "IN_REVIEW"
	ContentFeedbackStatusApproved ContentFeedbackStatus = "APPROVED"
	ContentFeedbackStatusRejected ContentFeedbackStatus = "REJECTED"
	ContentFeedbackStatusResolved ContentFeedbackStatus = "RESOLVED"
)

// ContentFeedback records a user report or request. It never changes content automatically.
type ContentFeedback struct {
	ID               int64
	MenuOptionID     *int64
	ResponseGroupID  *int64
	FeedbackType     ContentFeedbackType
	Category         ContentFeedbackCategory
	Subject          *string
	Message          string
	SuggestedContent *string
	SourceURL        *string
	Status           ContentFeedbackStatus
	AdminNotes       *string
	CreatedAt        time.Time
	ReviewedAt       *time.Time
	ResolvedAt       *time.Time

	MenuOption *MenuOption
}
