package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"calbot/internal/domain"
)

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) CreateStaff(ctx context.Context, username, passwordHash string) (domain.User, error) {
	const query = `INSERT INTO users (username, password_hash, role)
VALUES ($1, $2, 'STAFF')
RETURNING id, username, password_hash, role, created_at`
	return scanUser(s.pool.QueryRow(ctx, query, username, passwordHash))
}

func (s *UserStore) CreateInitialAdmin(ctx context.Context, username, passwordHash string) (domain.User, bool, error) {
	const query = `INSERT INTO users (username, password_hash, role)
SELECT $1, $2, 'ADMIN'
WHERE NOT EXISTS (SELECT 1 FROM users WHERE role = 'ADMIN')
RETURNING id, username, password_hash, role, created_at`
	user, err := scanUser(s.pool.QueryRow(ctx, query, username, passwordHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, false, nil
	}
	return user, err == nil, err
}

func (s *UserStore) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	const query = `SELECT id, username, password_hash, role, created_at FROM users WHERE username = $1`
	return scanUser(s.pool.QueryRow(ctx, query, username))
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	return user, err
}
