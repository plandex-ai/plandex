package utils

import (
	"testing"
)

func TestEnsureValidPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/", "/home/user"},
		{"/", "/"},
	}

	for _, test := range tests {
		t.Run("Path: "+test.input, func(t *testing.T) {
			output := EnsureValidPath(test.input)
			// t.Logf(output)
			if output != test.expected {
				t.Errorf("EnsureValidPath(%q) = %q; want %q", test.input, output, test.expected)
			}
		})
	}
}
