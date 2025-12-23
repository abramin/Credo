package models

import "strings"

// SanitizeKeySegment escapes delimiter characters in rate limit key segments
// to prevent key collision attacks where user-controlled identifiers containing
// ':' could manipulate adjacent rate limit buckets.
//
// Example: An identifier "user:admin" would become "user_admin", preventing
// it from being interpreted as a separate key segment.
func SanitizeKeySegment(s string) string {
	return strings.ReplaceAll(s, ":", "_")
}
