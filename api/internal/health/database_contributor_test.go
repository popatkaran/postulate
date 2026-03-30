package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/popatkaran/postulate/api/internal/health"
)

// fakePinger is a test double for the pool used by DatabaseContributor.
type fakePinger struct {
	pingErr error
	block   bool // if true, Ping blocks until ctx is cancelled
}

func (f *fakePinger) Ping(ctx context.Context) error {
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return f.pingErr
}

func (f *fakePinger) Stat() *pgxpool.Stat { return nil }

var _ health.PoolPinger = (*fakePinger)(nil) // compile-time interface check

func TestDatabaseContributor_Name(t *testing.T) {
	c := health.NewDatabaseContributorForTest(&fakePinger{})
	if c.Name() != "database" {
		t.Errorf("expected name 'database', got %q", c.Name())
	}
}

func TestDatabaseContributor_HealthyPing_ReturnsHealthyWithStats(t *testing.T) {
	c := health.NewDatabaseContributorForTest(&fakePinger{})
	result := c.Check(context.Background())

	if result.Status != health.StatusHealthy {
		t.Errorf("expected healthy, got %s", result.Status)
	}
	if result.Extensions == nil {
		t.Fatal("expected Extensions to be non-nil on healthy result")
	}
	if _, ok := result.Extensions["stats"]; !ok {
		t.Error("expected 'stats' key in Extensions")
	}
}

func TestDatabaseContributor_FailedPing_ReturnsUnhealthyWithMessage(t *testing.T) {
	pingErr := errors.New("connection refused")
	c := health.NewDatabaseContributorForTest(&fakePinger{pingErr: pingErr})
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
	if result.Message != pingErr.Error() {
		t.Errorf("expected message %q, got %q", pingErr.Error(), result.Message)
	}
	if result.Extensions != nil {
		t.Error("expected Extensions to be nil on unhealthy result")
	}
}

func TestDatabaseContributor_PingTimeout_ReturnsUnhealthyWithTimeoutMessage(t *testing.T) {
	c := health.NewDatabaseContributorForTest(&fakePinger{block: true})
	result := c.Check(context.Background())

	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
	if result.Message != "ping timeout" {
		t.Errorf("expected message 'ping timeout', got %q", result.Message)
	}
}
