package user

import (
	"context"

	"calbot/internal/domain"
)

// Store contains only the persistence operations required by authentication.
type Store interface {
	CreateStaff(ctx context.Context, username, passwordHash string) (domain.User, error)
	CreateInitialAdmin(ctx context.Context, username, passwordHash string) (domain.User, bool, error)
	FindByUsername(ctx context.Context, username string) (domain.User, error)
}
