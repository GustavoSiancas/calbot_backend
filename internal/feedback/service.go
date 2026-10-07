package feedback

import (
	"context"
	"errors"
	"strings"

	"calbot/internal/domain"
)

var ErrInvalidFeedback = errors.New("invalid content feedback")

type CreateInput struct {
	MenuOptionID     *int64
	ResponseGroupID  *int64
	FeedbackType     domain.ContentFeedbackType
	Category         domain.ContentFeedbackCategory
	Subject          *string
	Message          string
	SuggestedContent *string
	SourceURL        *string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create stores a report for later staff review. It does not modify menu content.
func (s *Service) Create(ctx context.Context, input CreateInput) (domain.ContentFeedback, error) {
	if strings.TrimSpace(input.Message) == "" || !isValidFeedbackType(input.FeedbackType) || !isValidCategory(input.Category) {
		return domain.ContentFeedback{}, ErrInvalidFeedback
	}
	return s.store.Create(ctx, domain.ContentFeedback{
		MenuOptionID:     input.MenuOptionID,
		ResponseGroupID:  input.ResponseGroupID,
		FeedbackType:     input.FeedbackType,
		Category:         input.Category,
		Subject:          input.Subject,
		Message:          strings.TrimSpace(input.Message),
		SuggestedContent: input.SuggestedContent,
		SourceURL:        input.SourceURL,
		Status:           domain.ContentFeedbackStatusPending,
	})
}

func isValidFeedbackType(value domain.ContentFeedbackType) bool {
	return value == domain.ContentFeedbackTypeContentReport || value == domain.ContentFeedbackTypeContentRequest
}

func isValidCategory(value domain.ContentFeedbackCategory) bool {
	switch value {
	case domain.ContentFeedbackCategoryIncorrect,
		domain.ContentFeedbackCategoryIncomplete,
		domain.ContentFeedbackCategoryMissing,
		domain.ContentFeedbackCategoryUnclear,
		domain.ContentFeedbackCategoryOutdated,
		domain.ContentFeedbackCategoryOther:
		return true
	default:
		return false
	}
}
