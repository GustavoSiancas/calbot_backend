package user

import (
	"context"
	"errors"
	"strings"

	"calbot/internal/domain"
	"calbot/internal/security"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInitialAdminExists = errors.New("initial admin already exists")
)

type Service struct {
	store Store
	jwt   *security.JWTService
}

func NewService(store Store, jwt *security.JWTService) *Service {
	return &Service{store: store, jwt: jwt}
}

func (s *Service) CreateInitialAdmin(ctx context.Context, username, password string) (domain.User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return domain.User{}, errors.New("username and password are required")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	user, created, err := s.store.CreateInitialAdmin(ctx, strings.TrimSpace(username), hash)
	if err != nil {
		return domain.User{}, err
	}
	if !created {
		return domain.User{}, ErrInitialAdminExists
	}
	return user, nil
}

func (s *Service) RegisterStaff(ctx context.Context, adminToken, username, password string) (domain.User, error) {
	claims, err := s.jwt.Validate(adminToken)
	if err != nil {
		return domain.User{}, ErrUnauthorized
	}
	if claims.Role != domain.UserRoleAdmin {
		return domain.User{}, ErrForbidden
	}
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return domain.User{}, errors.New("username and password are required")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	return s.store.CreateStaff(ctx, strings.TrimSpace(username), hash)
}

func (s *Service) Login(ctx context.Context, username, password string) (string, domain.User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return "", domain.User{}, ErrUnauthorized
	}

	account, err := s.store.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return "", domain.User{}, ErrUnauthorized
	}

	valid, err := security.VerifyPassword(password, account.PasswordHash)
	if err != nil || !valid {
		return "", domain.User{}, ErrUnauthorized
	}
	token, err := s.jwt.Issue(account.Username, account.Role)
	if err != nil {
		return "", domain.User{}, err
	}
	return token, account, nil
}
