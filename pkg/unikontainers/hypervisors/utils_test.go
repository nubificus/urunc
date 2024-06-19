package hypervisors

import (
	"testing"
)

func TestAppendNonEmpty(t *testing.T) {
	tests := []struct {
		body     string
		prefix   string
		value    string
		expected string
	}{
		{"hello", ", ", "world", "hello, world"},
		{"foo", "-", "bar", "foo-bar"},
		{"test", ":", "", "test"},
		{"", "", "value", "value"},
		{"base", " + ", "addition", "base + addition"},
		{"unchanged", "|", "", "unchanged"},
	}

	for _, tt := range tests {
		result := appendNonEmpty(tt.body, tt.prefix, tt.value)
		if result != tt.expected {
			t.Errorf("appendNonEmpty(%q, %q, %q) = %v, want %v", tt.body, tt.prefix, tt.value, result, tt.expected)
		}
	}
}
