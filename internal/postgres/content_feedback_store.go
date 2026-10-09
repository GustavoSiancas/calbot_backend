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
	full_name, ip, message
) VALUES ($1, $2, $3)
RETURNING id, full_name, host(ip), message, created_at`

	err := s.pool.QueryRow(ctx, query,
		feedback.FullName,
		feedback.IP,
		feedback.Message,
	).Scan(
		&feedback.ID,
		&feedback.FullName,
		&feedback.IP,
		&feedback.Message,
		&feedback.CreatedAt,
	)
	return feedback, err
}
