package security

import (
	"testing"
	"time"

	"calbot/internal/domain"
)

func TestJWTServiceIssueAndValidate(t *testing.T) {
	service, err := NewJWTService("a-secure-secret-with-at-least-32-bytes", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTService() error = %v", err)
	}
	service.now = func() time.Time { return time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC) }

	token, err := service.Issue("staff-user", domain.UserRoleStaff)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	claims, err := service.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.Username != "staff-user" {
		t.Errorf("claims.Username = %q; want %q", claims.Username, "staff-user")
	}
	if claims.Role != domain.UserRoleStaff {
		t.Errorf("claims.Role = %q; want %q", claims.Role, domain.UserRoleStaff)
	}
}
