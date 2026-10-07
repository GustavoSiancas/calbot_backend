package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"calbot/internal/domain"
)

type ContentFeedbackStore struct {
	pool *pgxpool.Pool
}

func NewContentFeedbackStore(pool *pgxpool.Pool) *ContentFeedbackStore {
	return &ContentFeedbackStore{pool: pool}
}

func (s *ContentFeedbackStore) Create(ctx context.Context, feedback domain.ContentFeedback) (domain.ContentFeedback, error) {
	const query = `INSERT INTO content_feedback (
	menu_option_id, response_group_id, feedback_type, category,
	subject, message, suggested_content, source_url, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PENDING')
RETURNING id, menu_option_id, response_group_id, feedback_type, category,
subject, message, suggested_content, source_url, status, admin_notes, created_at, reviewed_at, resolved_at`

	err := s.pool.QueryRow(ctx, query,
		feedback.MenuOptionID,
		feedback.ResponseGroupID,
		feedback.FeedbackType,
		feedback.Category,
		feedback.Subject,
		feedback.Message,
		feedback.SuggestedContent,
		feedback.SourceURL,
	).Scan(
		&feedback.ID,
		&feedback.MenuOptionID,
		&feedback.ResponseGroupID,
		&feedback.FeedbackType,
		&feedback.Category,
		&feedback.Subject,
		&feedback.Message,
		&feedback.SuggestedContent,
		&feedback.SourceURL,
		&feedback.Status,
		&feedback.AdminNotes,
		&feedback.CreatedAt,
		&feedback.ReviewedAt,
		&feedback.ResolvedAt,
	)
	return feedback, err
}
