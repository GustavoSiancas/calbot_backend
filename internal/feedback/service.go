package feedback

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"calbot/internal/domain"
)

var ErrInvalidFeedback = errors.New("invalid content feedback")

type CreateInput struct {
	FullName string
	IP       string
	Message  string
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create stores a report for later staff review. It does not modify menu content.
func (s *Service) Create(ctx context.Context, input CreateInput) (domain.ContentFeedback, error) {
	if strings.TrimSpace(input.FullName) == "" || strings.TrimSpace(input.Message) == "" {
		return domain.ContentFeedback{}, ErrInvalidFeedback
	}
	if _, err := netip.ParseAddr(strings.TrimSpace(input.IP)); err != nil {
		return domain.ContentFeedback{}, ErrInvalidFeedback
	}
	return s.store.Create(ctx, domain.ContentFeedback{
		FullName: strings.TrimSpace(input.FullName),
		IP:       strings.TrimSpace(input.IP),
		Message:  strings.TrimSpace(input.Message),
	})
}
