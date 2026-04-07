// Package domain contains the core business types for the Postulate platform.
// It has no external dependencies — no database drivers, no HTTP types.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the access level of a user.
type UserRole string

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	RolePlatformMember UserRole = "platform_member"
	RolePlatformAdmin  UserRole = "platform_admin"
)

const (
	StatusActive              UserStatus = "active"
	StatusSuspended           UserStatus = "suspended"
	StatusPendingVerification UserStatus = "pending_verification"
)

// User is the core identity entity.
// PasswordHash is nil for OAuth-only users; the column is retained for future
// email/password support but is not required.
type User struct {
	ID            uuid.UUID
	Email         string
	EmailVerified bool
	PasswordHash  *string
	FullName      string
	Role          UserRole
	Status        UserStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}
