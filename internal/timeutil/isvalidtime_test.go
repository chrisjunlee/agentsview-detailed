package timeutil

import "testing"

func TestIsValidTime(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		// Valid HH:MM values.
		{"00:00", true},
		{"09:15", true},
		{"23:59", true},

		// Hour out of range.
		{"24:00", false},
		// Minute out of range.
		{"12:60", false},
		// Hour is only one digit (len != 5).
		{"1:30", false},
		// Three-digit hour (len != 5).
		{"120:00", false},
		// Minute is only one digit (len != 5).
		{"12:3", false},
		// Seconds suffix makes it too long.
		{"12:30:00", false},
		// Non-numeric components.
		{"ab:cd", false},
		// Empty string.
		{"", false},
		// Wrong separator.
		{"12-30", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			if got := IsValidTime(tt.in); got != tt.want {
				t.Errorf("IsValidTime(%q) = %v, want %v",
					tt.in, got, tt.want)
			}
		})
	}
}
