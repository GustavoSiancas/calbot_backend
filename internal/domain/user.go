package domain

import "time"

type UserRole string

const (
	UserRoleStaff UserRole = "STAFF"
	UserRoleAdmin UserRole = "ADMIN"
)

// User is an account authorized to access protected administration services.
// PasswordHash must only contain a derived password value, never a plain password.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
}
