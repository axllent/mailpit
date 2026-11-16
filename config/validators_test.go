package config

import (
	"strings"
	"testing"
)

func TestParseMaxAge(t *testing.T) {
	tests := []struct {
		value     string
		wantHours int
		wantErr   string
	}{
		{"", 0, ""},
		{"24h", 24, ""},
		{"7d", 168, ""},
		{"0d", 0, ""},
		{"abc", 0, "must be either"},
		{"7w", 0, "must be either"},

		// largest values that fit in a time.Duration (controls for the cases below)
		{"99999d", 2399976, ""},
		{"2562047h", 2562047, ""},
		{"106751d", 2562024, ""},

		// one past the limit, and values that previously overflowed in the prune job
		// and deleted every message (or wrapped negative and disabled pruning)
		{"2562048h", 0, "too large"},
		{"106752d", 0, "too large"},
		{"999999d", 0, "too large"},
		{"9999999h", 0, "too large"},
		{"384307168202282326d", 0, "too large"},
		{"99999999999999999999h", 0, "too large"},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			origAge, origHours := MaxAge, MaxAgeInHours
			t.Cleanup(func() { MaxAge, MaxAgeInHours = origAge, origHours })

			MaxAge, MaxAgeInHours = tc.value, 0

			err := parseMaxAge()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want error containing %q", err, tc.wantErr)
			}
			if MaxAgeInHours != tc.wantHours {
				t.Errorf("MaxAgeInHours = %d, want %d", MaxAgeInHours, tc.wantHours)
			}
		})
	}
}
