package config

import (
	"testing"
)

func TestIsColorDark(t *testing.T) {
	tests := []struct {
		name     string
		color    string
		expected bool
	}{
		// Dark colors
		{
			name:     "black",
			color:    "rgb:0000/0000/0000",
			expected: true,
		},
		{
			name:     "dark gray",
			color:    "rgb:2020/2020/2020",
			expected: true,
		},
		{
			name:     "dark blue",
			color:    "rgb:0000/0000/3030",
			expected: true,
		},
		// Light colors
		{
			name:     "white",
			color:    "rgb:ffff/ffff/ffff",
			expected: false,
		},
		{
			name:     "light gray",
			color:    "rgb:e0e0/e0e0/e0e0",
			expected: false,
		},
		{
			name:     "light yellow",
			color:    "rgb:ffff/ffff/d0d0",
			expected: false,
		},
		// Edge cases
		{
			name:     "middle gray",
			color:    "rgb:8080/8080/8080",
			expected: false, // Luminance should be > 0.5
		},
		{
			name:     "invalid format",
			color:    "not-a-color",
			expected: true, // Default to dark on error (from IsColorDark implementation)
		},
		{
			name:     "empty string",
			color:    "",
			expected: true, // Default to dark on error (from IsColorDark implementation)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsColorDark(tt.color)
			if result != tt.expected {
				t.Errorf("IsColorDark(%q) = %v, want %v", tt.color, result, tt.expected)
			}
		})
	}
}

