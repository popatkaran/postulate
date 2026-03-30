//go:build integration

package migrate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/popatkaran/postulate/api/internal/config"
	"github.com/popatkaran/postulate/api/internal/database"
	apimigrate "github.com/popatkaran/postulate/api/internal/migrate"
)

// schemaPool returns a pool connected to postulate_test with all migrations applied.
func schemaPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := config.DatabaseConfig{
		Host: "localhost", Port: 5432, Name: "postulate_test",
		User: "postulate_dev", Password: "postulate_dev", SSLMode: "disable",
		MaxOpenConns: 5, MaxIdleConns: 1, ConnMaxLifetimeSeconds: 60,
	}
	pool, err := database.New(context.Background(), cfg, integrationLogger)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	if err := apimigrate.Run(context.Background(), pool, integrationLogger); err != nil {
		pool.Close()
		t.Fatalf("migrate up: %v", err)
	}
	return pool
}

func TestSchema_TablesExist(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	for _, table := range []string{"users", "sessions", "refresh_tokens"} {
		var exists bool
		err := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1 AND table_schema = 'public')",
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %q to exist", table)
		}
	}
}

func TestSchema_UsersColumns(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	expected := []string{"id", "email", "email_verified", "password_hash", "full_name", "role", "status", "created_at", "updated_at", "deleted_at"}
	assertColumns(t, pool, "users", expected)
}

func TestSchema_SessionsColumns(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	expected := []string{"id", "user_id", "token_hash", "ip_address", "user_agent", "last_active_at", "expires_at", "created_at", "revoked_at"}
	assertColumns(t, pool, "sessions", expected)
}

func TestSchema_RefreshTokensColumns(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	expected := []string{"id", "session_id", "user_id", "token_hash", "expires_at", "used_at", "created_at"}
	assertColumns(t, pool, "refresh_tokens", expected)
}

func TestSchema_IndexesExist(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	indexes := []string{
		"idx_users_email", "idx_users_status", "idx_users_deleted_at",
		"idx_sessions_user_id", "idx_sessions_token_hash", "idx_sessions_expires_at",
		"idx_refresh_tokens_session_id", "idx_refresh_tokens_user_id",
		"idx_refresh_tokens_token_hash", "idx_refresh_tokens_expires_at",
	}
	for _, idx := range indexes {
		var exists bool
		err := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)", idx,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("query index %s: %v", idx, err)
		}
		if !exists {
			t.Errorf("expected index %q to exist", idx)
		}
	}
}

func TestSchema_CascadeDelete_UserDeletesCascadesToSessionsAndTokens(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Insert user
	var userID string
	err := pool.QueryRow(ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		"cascade@example.com", "hash",
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	// Insert session
	var sessionID string
	err = pool.QueryRow(ctx,
		"INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, NOW() + INTERVAL '1 hour') RETURNING id",
		userID, "session-token-hash",
	).Scan(&sessionID)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// Insert refresh token
	_, err = pool.Exec(ctx,
		"INSERT INTO refresh_tokens (session_id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, NOW() + INTERVAL '1 hour')",
		sessionID, userID, "refresh-token-hash",
	)
	if err != nil {
		t.Fatalf("insert refresh_token: %v", err)
	}

	// Delete user — should cascade
	_, err = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Fatalf("delete user: %v", err)
	}

	// Verify sessions and refresh_tokens are gone
	var count int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE id = $1", sessionID).Scan(&count) //nolint:errcheck
	if count != 0 {
		t.Error("expected session to be cascade-deleted")
	}
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM refresh_tokens WHERE session_id = $1", sessionID).Scan(&count) //nolint:errcheck
	if count != 0 {
		t.Error("expected refresh_tokens to be cascade-deleted")
	}
}

func TestSchema_UsersEmailUniqueness(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2)", "unique@example.com", "hash1")
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2)", "unique@example.com", "hash2")
	assertPgError(t, err, "23505", "expected unique violation on users.email")

	// Cleanup
	pool.Exec(ctx, "DELETE FROM users WHERE email = 'unique@example.com'") //nolint:errcheck
}

func TestSchema_UsersRoleCheckConstraint(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	_, err := pool.Exec(context.Background(),
		"INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)",
		"role@example.com", "hash", "superuser",
	)
	assertPgError(t, err, "23514", "expected check violation on users.role")
}

func TestSchema_UsersStatusCheckConstraint(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	_, err := pool.Exec(context.Background(),
		"INSERT INTO users (email, password_hash, status) VALUES ($1, $2, $3)",
		"status@example.com", "hash", "deleted",
	)
	assertPgError(t, err, "23514", "expected check violation on users.status")
}

func TestSchema_SessionsForeignKeyConstraint(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	_, err := pool.Exec(context.Background(),
		"INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, NOW() + INTERVAL '1 hour')",
		"00000000-0000-0000-0000-000000000000", "fk-test-hash",
	)
	assertPgError(t, err, "23503", "expected FK violation on sessions.user_id")
}

func TestSchema_RefreshTokensForeignKeyConstraint(t *testing.T) {
	pool := schemaPool(t)
	defer pool.Close()

	_, err := pool.Exec(context.Background(),
		"INSERT INTO refresh_tokens (session_id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, NOW() + INTERVAL '1 hour')",
		"00000000-0000-0000-0000-000000000000",
		"00000000-0000-0000-0000-000000000000",
		"fk-refresh-hash",
	)
	assertPgError(t, err, "23503", "expected FK violation on refresh_tokens.session_id")
}

// assertColumns checks that all expected columns exist on the given table.
func assertColumns(t *testing.T, pool *pgxpool.Pool, table string, expected []string) {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		"SELECT column_name FROM information_schema.columns WHERE table_name = $1 AND table_schema = 'public'",
		table,
	)
	if err != nil {
		t.Fatalf("query columns for %s: %v", table, err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("scan: %v", err)
		}
		found[col] = true
	}
	for _, col := range expected {
		if !found[col] {
			t.Errorf("table %q: expected column %q", table, col)
		}
	}
}

// assertPgError checks that err is a *pgconn.PgError with the given SQLSTATE code.
func assertPgError(t *testing.T, err error, code string, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error, got nil", msg)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("%s: expected *pgconn.PgError, got %T: %v", msg, err, err)
	}
	if pgErr.Code != code {
		t.Errorf("%s: expected SQLSTATE %s, got %s", msg, code, pgErr.Code)
	}
}
