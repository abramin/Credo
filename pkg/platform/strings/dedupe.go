// Package strings provides string manipulation utilities.
package strings

import (
	"strings"
)

// DedupeAndTrim removes duplicates and empty strings from a slice,
// trimming whitespace from each element. Order is preserved.
//
// Example:
//
//	DedupeAndTrim([]string{"  foo ", "bar", "foo", "", "  "})
//	// Returns: []string{"foo", "bar"}
func DedupeAndTrim(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; !ok {
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}

	return result
}

// DedupeAndTrimLower is like DedupeAndTrim but also lowercases each element.
// Useful for case-insensitive deduplication.
//
// Example:
//
//	DedupeAndTrimLower([]string{"  FOO ", "bar", "Foo"})
//	// Returns: []string{"foo", "bar"}
func DedupeAndTrimLower(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, v := range values {
		trimmed := strings.ToLower(strings.TrimSpace(v))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; !ok {
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}

	return result
}
