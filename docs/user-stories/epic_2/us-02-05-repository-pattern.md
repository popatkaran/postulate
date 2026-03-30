# US-02-05 — Repository Pattern Foundation

**Epic:** Epic 02 — Database Schema and Migration Tooling
**Version:** 1.0.0
**Status:** Ready for Development
**Priority:** Must Have
**Depends on:** US-02-04 (schema in place), US-02-01 (pool available)

---

## 1. Story

As a **platform engineer**, I need a consistent repository pattern with base interfaces, transaction support, and context propagation established so that all data access code in Epic 03 and beyond follows the same structure, is independently testable, and never leaks database implementation details into business logic.

---

## 2. Background

The repository pattern separates data access logic from business logic. Each entity (user, session, refresh token) has a repository interface defined in terms of the application's domain types and a concrete implementation that uses `pgxpool` to execute SQL.

Business logic — authentication handlers, generation engine, project management — depends only on repository interfaces, never on concrete implementations or `pgx` directly. This makes business logic independently unit-testable using mock repositories, without requiring a database connection.

This story establishes the pattern, the shared infrastructure it requires, and the three concrete repository implementations for the entities defined in US-02-04. Epic 03 will use these repositories directly — it adds no database code of its own.

---

## 3. Domain Types

Domain types live in `api/internal/domain/` and are plain Go structs — no `pgx` types, no `database/sql` types, no JSON tags for database serialisation. They are the language of the application, not the language of the database.

```go
// api/internal/domain/user.go
type User struct {
    ID            uuid.UUID
    Email         string
    EmailVerified bool
    PasswordHash  string
    FullName      string
    Role          UserRole
    Status        UserStatus
    CreatedAt     time.Time
    UpdatedAt     time.Time
    DeletedAt     *time.Time
}

type UserRole   string
type UserStatus string

const (
    RoleMember        UserRole = "member"
    RoleAdmin         UserRole = "admin"
    RolePlatformAdmin UserRole = "platform_admin"
)

const (
    StatusActive              UserStatus = "active"
    StatusSuspended           UserStatus = "suspended"
    StatusPendingVerification UserStatus = "pending_verification"
)
```

```go
// api/internal/domain/session.go
type Session struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    TokenHash    string
    IPAddress    string
    UserAgent    string
    LastActiveAt time.Time
    ExpiresAt    time.Time
    CreatedAt    time.Time
    RevokedAt    *time.Time
}
```

```go
// api/internal/domain/refresh_token.go
type RefreshToken struct {
    ID        uuid.UUID
    SessionID uuid.UUID
    UserID    uuid.UUID
    TokenHash string
    ExpiresAt time.Time
    UsedAt    *time.Time
    CreatedAt time.Time
}
```

---

## 4. Repository Interfaces

```go
// api/internal/repository/user_repository.go
type UserRepository interface {
    Create(ctx context.Context, user *domain.User) error
    FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    FindByEmail(ctx context.Context, email string) (*domain.User, error)
    Update(ctx context.Context, user *domain.User) error
    SoftDelete(ctx context.Context, id uuid.UUID) error
}

// api/internal/repository/session_repository.go
type SessionRepository interface {
    Create(ctx context.Context, session *domain.Session) error
    FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error)
    UpdateLastActive(ctx context.Context, id uuid.UUID, at time.Time) error
    Revoke(ctx context.Context, id uuid.UUID) error
    RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
    DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// api/internal/repository/refresh_token_repository.go
type RefreshTokenRepository interface {
    Create(ctx context.Context, token *domain.RefreshToken) error
    FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
    MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
    DeleteBySessionID(ctx context.Context, sessionID uuid.UUID) error
    DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}
```

---

## 5. Transaction Support

Some operations in Epic 03 require atomicity across multiple repository calls — for example, creating a session and its initial refresh token together. The transaction pattern uses a `Transactor` interface:

```go
// api/internal/repository/transaction.go
type Transactor interface {
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

Concrete repository implementations that need transaction support accept a `pgxpool.Pool` and implement `Transactor`. The transaction context carries the active `pgx.Tx` — repository methods check the context for an active transaction and use it if present, otherwise fall back to the pool.

---

## 6. Acceptance Criteria

1. Domain types for `User`, `Session`, and `RefreshToken` are defined in `api/internal/domain/` as plain Go structs with no external library dependencies.
2. Repository interfaces for all three entities are defined in `api/internal/repository/` as Go interfaces.
3. Concrete `pgx` implementations exist for all three interfaces in `api/internal/repository/postgres/`.
4. All repository method implementations:
   - Accept `context.Context` as their first argument
   - Use named column lists — no `SELECT *`
   - Return `domain.ErrNotFound` (a sentinel error) when a record is not found — not `pgx.ErrNoRows` directly
   - Return `domain.ErrConflict` when a unique constraint is violated — not the raw `pgconn.PgError` directly
5. The `Transactor` interface is implemented — `WithTransaction` begins a `pgx` transaction, calls `fn`, commits on nil return, rolls back on error.
6. Repository implementations can operate within a transaction when one is present in the context.
7. No `pgx` types leak into the `domain` or `repository` (interface) packages — they are confined to `repository/postgres/`.
8. The `uuid` package used is `github.com/google/uuid` — consistent across domain types and repository implementations.
9. Unit tests for each repository use a mock implementation of the interface — no database required.
10. Integration tests for each repository use the `postulate_test` database and verify all interface methods.
11. Minimum 90% coverage across all packages introduced in this story.

---

## 7. Error Sentinel Values

```go
// api/internal/domain/errors.go
var (
    ErrNotFound = errors.New("record not found")
    ErrConflict = errors.New("record already exists")
)
```

Repository implementations map `pgx` and `pgconn` errors to these sentinels. Business logic catches the sentinels — never the raw database errors.

---

## 8. Tasks

### Task 1 — Add uuid dependency
- Add `github.com/google/uuid` to `api/go.mod`
- Run `go mod tidy`

### Task 2 — Define domain types
- Create `api/internal/domain/user.go` with `User`, `UserRole`, `UserStatus` types and constants per Section 3
- Create `api/internal/domain/session.go` with `Session` type
- Create `api/internal/domain/refresh_token.go` with `RefreshToken` type
- Create `api/internal/domain/errors.go` with `ErrNotFound` and `ErrConflict` sentinel errors

### Task 3 — Define repository interfaces
- Create `api/internal/repository/user_repository.go` with `UserRepository` interface per Section 4
- Create `api/internal/repository/session_repository.go` with `SessionRepository` interface per Section 4
- Create `api/internal/repository/refresh_token_repository.go` with `RefreshTokenRepository` interface per Section 4
- Create `api/internal/repository/transaction.go` with `Transactor` interface per Section 5

### Task 4 — Implement UserRepository
- Create `api/internal/repository/postgres/user_repository.go`
- Implement all five methods: `Create`, `FindByID`, `FindByEmail`, `Update`, `SoftDelete`
- `Create` uses `INSERT ... RETURNING` to populate the `id`, `created_at`, `updated_at` fields
- `FindByID` and `FindByEmail` return `domain.ErrNotFound` on `pgx.ErrNoRows`
- `Create` maps `pgconn.PgError` code `23505` (unique violation) to `domain.ErrConflict`
- `SoftDelete` sets `deleted_at = NOW()` and `updated_at = NOW()`

### Task 5 — Implement SessionRepository
- Create `api/internal/repository/postgres/session_repository.go`
- Implement all seven methods per the interface in Section 4
- `DeleteExpired` returns the count of deleted rows
- `RevokeAllForUser` sets `revoked_at = NOW()` on all non-revoked sessions for the user

### Task 6 — Implement RefreshTokenRepository
- Create `api/internal/repository/postgres/refresh_token_repository.go`
- Implement all five methods per the interface in Section 4
- `MarkUsed` sets `used_at = NOW()`
- `DeleteExpired` returns the count of deleted rows

### Task 7 — Implement Transactor
- Create `api/internal/repository/postgres/transactor.go`
- Implement `PostgresTransactor` struct accepting `*pgxpool.Pool`
- Implement `WithTransaction`: begin tx, store in context, call `fn(ctx)`, commit or rollback
- Define a typed context key for the transaction — same pattern as request ID in US-01-07
- Create `TxFromContext(ctx) pgx.Tx` and `ContextWithTx(ctx, tx) context.Context` helpers
- Repository methods check `TxFromContext` and use the transaction if present

### Task 8 — Unit tests with mocks
- Create `api/internal/repository/mock/` directory
- Generate or hand-write mock implementations of all three repository interfaces
- Create unit tests in `api/internal/repository/postgres/*_test.go` using `pgxmock` to verify SQL correctness:
  - Correct SQL statement executed for each method
  - Correct argument binding
  - `ErrNotFound` returned on no rows
  - `ErrConflict` returned on unique violation

### Task 9 — Integration tests
- Create `api/internal/repository/postgres/*_integration_test.go` with `//go:build integration` tag
- For `UserRepository`:
  - Test: `Create` inserts a user and returns the populated struct
  - Test: `FindByEmail` returns the created user
  - Test: `Create` with duplicate email returns `domain.ErrConflict`
  - Test: `FindByID` with unknown ID returns `domain.ErrNotFound`
  - Test: `SoftDelete` sets `deleted_at` and does not physically remove the row
- For `SessionRepository`:
  - Test: `Create` inserts a session for an existing user
  - Test: `Revoke` sets `revoked_at`
  - Test: `RevokeAllForUser` revokes all active sessions
  - Test: `DeleteExpired` removes sessions past their expiry
- For `RefreshTokenRepository`:
  - Test: `Create` inserts a token for an existing session
  - Test: `MarkUsed` sets `used_at`
  - Test: `DeleteBySessionID` removes all tokens for the session
- For `Transactor`:
  - Test: successful transaction commits both operations
  - Test: error from `fn` rolls back both operations — neither insert persists

---

## 9. Definition of Done

- All tasks completed
- All three repository interfaces implemented and tested
- `pgx` types confined to `repository/postgres/` — no leakage into domain or interface packages
- `domain.ErrNotFound` and `domain.ErrConflict` returned correctly — verified by tests
- Transaction support verified — rollback on error confirmed by integration test
- 90% coverage across all new packages
- All unit tests pass with `-race` flag
- Integration tests pass against `postulate_test` database
- `make lint` passes with zero issues
- PR reviewed and approved
- All acceptance criteria verified by reviewer
