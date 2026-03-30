package domain

import "errors"

// ErrNotFound is returned by repository methods when a record does not exist.
var ErrNotFound = errors.New("record not found")

// ErrConflict is returned by repository methods when a unique constraint is violated.
var ErrConflict = errors.New("record already exists")
