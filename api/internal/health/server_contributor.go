package health

import "context"

// ServerContributor reports the health of the server process itself.
// It returns healthy unconditionally — if the process can run this code, it is alive.
type ServerContributor struct{}

// Name returns the contributor identifier used in the health response.
func (s *ServerContributor) Name() string { return "server" }

// Check always returns healthy while the process is running.
func (s *ServerContributor) Check(_ context.Context) CheckResult {
	return CheckResult{Status: StatusHealthy}
}
