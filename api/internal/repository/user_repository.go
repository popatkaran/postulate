// Package repository defines the data access interfaces for the Postulate platform.
// Concrete implementations live in repository/postgres/.
// No pgx or database/sql types appear in this package.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// UserRepository defines all persistence operations for the User entity.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
