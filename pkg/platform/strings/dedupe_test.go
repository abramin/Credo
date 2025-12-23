package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedupeAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"foo"},
			expected: []string{"foo"},
		},
		{
			name:     "trims whitespace",
			input:    []string{"  foo  ", "bar  ", "  baz"},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "removes duplicates preserving order",
			input:    []string{"foo", "bar", "foo", "baz", "bar"},
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "removes empty strings",
			input:    []string{"foo", "", "  ", "bar"},
			expected: []string{"foo", "bar"},
		},
		{
			name:     "combined: trim, dedupe, remove empty",
			input:    []string{"  foo ", "bar", "foo", "", "  ", "bar"},
			expected: []string{"foo", "bar"},
		},
		{
			name:     "preserves case",
			input:    []string{"Foo", "foo", "FOO"},
			expected: []string{"Foo", "foo", "FOO"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DedupeAndTrim(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDedupeAndTrimLower(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: nil,
		},
		{
			name:     "lowercases and dedupes",
			input:    []string{"Foo", "foo", "FOO"},
			expected: []string{"foo"},
		},
		{
			name:     "trims, lowercases, and dedupes",
			input:    []string{"  FOO ", "bar", "Foo", "BAR"},
			expected: []string{"foo", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DedupeAndTrimLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
