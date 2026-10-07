package feedback

import (
	"context"

	"calbot/internal/domain"
)

type Store interface {
	Create(ctx context.Context, feedback domain.ContentFeedback) (domain.ContentFeedback, error)
}
