package health

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const pingTimeout = 2 * time.Second

// PoolPinger is the minimal interface DatabaseContributor needs from the pool.
// Defined here so tests can provide a fake without a real database.
type PoolPinger interface {
	Ping(ctx context.Context) error
	Stat() *pgxpool.Stat
}

// DatabaseContributor checks PostgreSQL reachability via a pool ping.
type DatabaseContributor struct {
	pool PoolPinger
}

// NewDatabaseContributor constructs a DatabaseContributor backed by the given pool.
// pool must satisfy PoolPinger — *pgxpool.Pool and database.Pool both do.
func NewDatabaseContributor(pool PoolPinger) *DatabaseContributor {
	return &DatabaseContributor{pool: pool}
}

// NewDatabaseContributorForTest constructs a DatabaseContributor with a custom
// PoolPinger. Intended for unit tests only.
func NewDatabaseContributorForTest(p PoolPinger) *DatabaseContributor {
	return &DatabaseContributor{pool: p}
}

// Name returns the contributor key used in the health response.
func (d *DatabaseContributor) Name() string { return "database" }

// Check pings the database with a 2-second timeout.
// On success it returns healthy with pool statistics in Extensions.
// On timeout it returns unhealthy with message "ping timeout".
// On any other error it returns unhealthy with the error message.
func (d *DatabaseContributor) Check(ctx context.Context) CheckResult {
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := d.pool.Ping(pingCtx); err != nil {
		msg := err.Error()
		if errors.Is(err, context.DeadlineExceeded) {
			msg = "ping timeout"
		}
		return CheckResult{Status: StatusUnhealthy, Message: msg}
	}

	stats := poolStatsFromPgx(d.pool.Stat())
	return CheckResult{
		Status: StatusHealthy,
		Extensions: map[string]any{
			"stats": stats,
		},
	}
}
