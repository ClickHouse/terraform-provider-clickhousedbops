package nilcompare

import (
	"testing"
)

func TestNilCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		// Both nil interface{} cases
		{
			name:     "both nil interfaces",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "first nil, second non-nil",
			a:        nil,
			b:        42,
			expected: false,
		},
		{
			name:     "first non-nil, second nil",
			a:        42,
			b:        nil,
			expected: false,
		},

		// Both pointers cases
		{
			name:     "both nil pointers",
			a:        (*int)(nil),
			b:        (*int)(nil),
			expected: true,
		},
		{
			name:     "first nil pointer, second non-nil pointer",
			a:        (*int)(nil),
			b:        new(42),
			expected: false,
		},
		{
			name:     "first non-nil pointer, second nil pointer",
			a:        new(42),
			b:        (*int)(nil),
			expected: false,
		},
		{
			name:     "both non-nil pointers with same values",
			a:        new(42),
			b:        new(42),
			expected: true,
		},
		{
			name:     "both non-nil pointers with different values",
			a:        new(42),
			b:        new(24),
			expected: false,
		},

		// Pointer vs scalar cases
		{
			name:     "nil pointer vs scalar",
			a:        (*int)(nil),
			b:        42,
			expected: false,
		},
		{
			name:     "scalar vs nil pointer",
			a:        42,
			b:        (*int)(nil),
			expected: false,
		},
		{
			name:     "non-nil pointer vs matching scalar",
			a:        new(42),
			b:        42,
			expected: true,
		},
		{
			name:     "non-nil pointer vs non-matching scalar",
			a:        new(42),
			b:        24,
			expected: false,
		},
		{
			name:     "scalar vs matching non-nil pointer",
			a:        42,
			b:        new(42),
			expected: true,
		},
		{
			name:     "scalar vs non-matching non-nil pointer",
			a:        42,
			b:        new(24),
			expected: false,
		},

		// Both scalars cases
		{
			name:     "equal scalars",
			a:        42,
			b:        42,
			expected: true,
		},
		{
			name:     "different scalars",
			a:        42,
			b:        24,
			expected: false,
		},

		// String type tests
		{
			name:     "equal strings",
			a:        "hello",
			b:        "hello",
			expected: true,
		},
		{
			name:     "different strings",
			a:        "hello",
			b:        "world",
			expected: false,
		},
		{
			name:     "string pointer vs string scalar",
			a:        new("hello"),
			b:        "hello",
			expected: true,
		},
		{
			name:     "nil string pointer vs string scalar",
			a:        (*string)(nil),
			b:        "hello",
			expected: false,
		},
		{
			name:     "nil string pointer vs string scalar",
			a:        (*string)(nil),
			b:        nil,
			expected: true,
		},

		// Boolean type tests
		{
			name:     "equal booleans",
			a:        true,
			b:        true,
			expected: true,
		},
		{
			name:     "different booleans",
			a:        true,
			b:        false,
			expected: false,
		},
		{
			name:     "bool pointer vs bool scalar",
			a:        new(true),
			b:        true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NilCompare(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("NilCompare(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
