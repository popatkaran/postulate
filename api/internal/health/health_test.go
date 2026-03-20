package health_test

import (
	"context"
	"testing"

	"github.com/popatkaran/postulate/api/internal/health"
)

// unhealthyContributor is a test double that always reports unhealthy.
type unhealthyContributor struct{ name string }

func (u *unhealthyContributor) Name() string { return u.name }
func (u *unhealthyContributor) Check(_ context.Context) health.CheckResult {
	return health.CheckResult{Status: health.StatusUnhealthy, Message: "test failure"}
}

func TestAggregator_AllHealthy_ReturnsHealthyStatus(t *testing.T) {
	// Arrange
	a := &health.Aggregator{}
	a.Register(&health.ServerContributor{})

	// Act
	result := a.Check(context.Background())

	// Assert
	if result.Status != health.StatusHealthy {
		t.Errorf("expected healthy, got %s", result.Status)
	}
}

func TestAggregator_AnyUnhealthy_ReturnsUnhealthyStatus(t *testing.T) {
	// Arrange
	a := &health.Aggregator{}
	a.Register(&health.ServerContributor{})
	a.Register(&unhealthyContributor{name: "db"})

	// Act
	result := a.Check(context.Background())

	// Assert
	if result.Status != health.StatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
}

func TestAggregator_ResultIncludesAllContributorNames(t *testing.T) {
	// Arrange
	a := &health.Aggregator{}
	a.Register(&health.ServerContributor{})
	a.Register(&unhealthyContributor{name: "cache"})

	// Act
	result := a.Check(context.Background())

	// Assert
	if _, ok := result.Checks["server"]; !ok {
		t.Error("expected checks to contain 'server'")
	}
	if _, ok := result.Checks["cache"]; !ok {
		t.Error("expected checks to contain 'cache'")
	}
	if result.Checks["cache"].Status != health.StatusUnhealthy {
		t.Errorf("expected cache to be unhealthy, got %s", result.Checks["cache"].Status)
	}
}
