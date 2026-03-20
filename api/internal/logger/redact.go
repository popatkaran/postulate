package logger

import (
	"strings"
)

// sensitiveKeyPatterns lists lowercase substrings that mark a field as sensitive.
var sensitiveKeyPatterns = []string{
	"password",
	"token",
	"secret",
	"key",
	"credential",
	"authorization",
}

// isSensitive reports whether the given attribute key should be redacted.
func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range sensitiveKeyPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}
