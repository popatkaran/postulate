package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/popatkaran/postulate/api/internal/database"
	"github.com/popatkaran/postulate/api/internal/domain"
)

// UserRepo is the pgx-backed implementation of repository.UserRepository.
type UserRepo struct{ pool database.Pool }

// NewUserRepo constructs a UserRepo.
func NewUserRepo(pool database.Pool) *UserRepo { return &UserRepo{pool: pool} }

// compile-time check that *pgxpool.Pool satisfies database.Pool
var _ database.Pool = (*pgxpool.Pool)(nil)

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	q := querier(ctx, r.pool)
	return mapErr(q.QueryRow(ctx,
		`INSERT INTO users (email, email_verified, password_hash, full_name, role, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		u.Email, u.EmailVerified, u.PasswordHash, u.FullName, u.Role, u.Status,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt))
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u := &domain.User{}
	err := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT id, email, email_verified, password_hash, full_name, role, status,
		        created_at, updated_at, deleted_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.EmailVerified, &u.PasswordHash, &u.FullName,
		&u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	return u, mapErr(err)
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u := &domain.User{}
	err := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT id, email, email_verified, password_hash, full_name, role, status,
		        created_at, updated_at, deleted_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.EmailVerified, &u.PasswordHash, &u.FullName,
		&u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	return u, mapErr(err)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE users SET email=$1, email_verified=$2, password_hash=$3, full_name=$4,
		        role=$5, status=$6, updated_at=NOW()
		 WHERE id=$7`,
		u.Email, u.EmailVerified, u.PasswordHash, u.FullName, u.Role, u.Status, u.ID,
	)
	return mapErr(err)
}

func (r *UserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`UPDATE users SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1`, id,
	)
	return mapErr(err)
}

// CountAll returns the total number of rows in the users table.
// This is used only by the bootstrap logic to detect a fresh installation.
// The query runs within any active transaction in ctx.
func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := querier(ctx, r.pool).QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, mapErr(err)
}
