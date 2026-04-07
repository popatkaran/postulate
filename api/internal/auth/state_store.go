package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const stateTTL = 5 * time.Minute

// StateStore is a short-lived, single-use store for OAuth state parameters.
// Each state value is valid for stateTTL and is deleted on first validation.
// This implementation is in-memory and suitable for single-instance deployments.
// Replace with a distributed store (e.g. Redis) for multi-instance deployments.
type StateStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

// NewStateStore constructs an empty StateStore.
func NewStateStore() *StateStore {
	return &StateStore{entries: make(map[string]time.Time)}
}

// Generate creates a cryptographically random state value, stores it with a
// 5-minute TTL, and returns it.
func (s *StateStore) Generate() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)
	s.mu.Lock()
	s.entries[state] = time.Now().Add(stateTTL)
	s.mu.Unlock()
	return state, nil
}

// Validate checks that state exists and has not expired, then deletes it
// (single-use). Returns false if the state is missing or expired.
func (s *StateStore) Validate(state string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.entries[state]
	if !ok {
		return false
	}
	delete(s.entries, state)
	return time.Now().Before(exp)
}
