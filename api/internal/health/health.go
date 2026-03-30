// Package health provides the health check aggregator and contributor interface.
package health

import (
	"context"
	"time"
)

// Status represents the health state of a contributor or the aggregate.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult is the result returned by a single Contributor.
type CheckResult struct {
	Status     Status         `json:"status"`
	Message    string         `json:"message"`
	Extensions map[string]any `json:"extensions,omitempty"`
}

// Contributor is implemented by any component that can report its health.
type Contributor interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// AggregateResult is the full health response body.
type AggregateResult struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// Aggregator collects named Contributor instances and produces an AggregateResult.
type Aggregator struct {
	contributors []Contributor
}

// Register adds a contributor to the aggregator.
func (a *Aggregator) Register(c Contributor) {
	a.contributors = append(a.contributors, c)
}

// Check calls every registered contributor and returns the aggregate result.
// The aggregate status is healthy only when all contributors are healthy.
func (a *Aggregator) Check(ctx context.Context) AggregateResult {
	checks := make(map[string]CheckResult, len(a.contributors))
	overall := StatusHealthy

	for _, c := range a.contributors {
		result := c.Check(ctx)
		checks[c.Name()] = result
		if result.Status != StatusHealthy {
			overall = StatusUnhealthy
		}
	}

	return AggregateResult{
		Status:    overall,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	}
}
