package models

import (
	"testing"
)

func TestIsValidID(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"123e4567-e89b-42d3-a456-426614174000", true},
		{"182e4689-fd89-4c2b-9ae5-4ea5ef0acab9", true},
		{"182E4689-FD89-4C2B-9AE5-4EA5EF0ACAB9", true},
		{"", false},
		{"123", false},
		{"123e4567e89b42d3a456426614174000", false},
		{"123e4567-e89b-42d3-a456-42661417400g", false},
		{"123e4567-e89b-02d3-a456-426614174000", false},
		{"123e4567-e89b-42d3-3256-426614174000", false},
		{"123e4567?e89b?42d3-a456-426614174000", false},
		{"00000000-0000-0000-0000-000000000000", false},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			if got := IsValidID(tt.val); got != tt.want {
				t.Errorf("IsValidID() = %v, want %v", got, tt.want)
			}
		})
	}
}
