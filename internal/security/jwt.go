package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"calbot/internal/domain"
)

var (
	ErrInvalidToken = errors.New("invalid JWT")
	ErrExpiredToken = errors.New("expired JWT")
)

type Claims struct {
	Username  string          `json:"sub"`
	Role      domain.UserRole `json:"role"`
	IssuedAt  int64           `json:"iat"`
	ExpiresAt int64           `json:"exp"`
}

// JWTService issues and validates signed HS256 access tokens for staff users.
type JWTService struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTService(secret string, ttl time.Duration) (*JWTService, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must contain at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("JWT TTL must be positive")
	}

	return &JWTService{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

func (s *JWTService) Issue(subject string, role domain.UserRole) (string, error) {
	if strings.TrimSpace(subject) == "" {
		return "", errors.New("token subject cannot be empty")
	}
	if role != domain.UserRoleStaff && role != domain.UserRoleAdmin {
		return "", errors.New("invalid user role")
	}

	now := s.now().UTC()
	claims := Claims{Username: subject, Role: role, IssuedAt: now.Unix(), ExpiresAt: now.Add(s.ttl).Unix()}
	header, err := encodeJWTPart(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", fmt.Errorf("encode JWT header: %w", err)
	}
	payload, err := encodeJWTPart(claims)
	if err != nil {
		return "", fmt.Errorf("encode JWT claims: %w", err)
	}

	signingInput := header + "." + payload
	return signingInput + "." + s.sign(signingInput), nil
}

func (s *JWTService) Validate(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return Claims{}, ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return Claims{}, ErrInvalidToken
	}

	expectedSignature := s.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var claims Claims
	if json.Unmarshal(payload, &claims) != nil || strings.TrimSpace(claims.Username) == "" || claims.ExpiresAt == 0 || (claims.Role != domain.UserRoleStaff && claims.Role != domain.UserRoleAdmin) {
		return Claims{}, ErrInvalidToken
	}
	if s.now().UTC().Unix() >= claims.ExpiresAt {
		return Claims{}, ErrExpiredToken
	}

	return claims, nil
}

func (s *JWTService) sign(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeJWTPart(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}
