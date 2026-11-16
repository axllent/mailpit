package storage

import (
	"math"
	"testing"
	"time"
)

func TestMaxAgeCutoff(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	maxHours := int(math.MaxInt64 / int64(time.Hour))

	tests := []struct {
		name   string
		hours  int
		want   time.Time
		wantOK bool
	}{
		{"disabled", 0, time.Time{}, false},
		{"negative", -1, time.Time{}, false},
		{"one day", 24, now.Add(-24 * time.Hour), true},
		// control: the largest representable age gives a cutoff in the past
		{"largest representable", maxHours, now.Add(-time.Duration(maxHours) * time.Hour), true},
		// one hour more overflowed and put the cutoff in the future, deleting every message
		{"one past largest", maxHours + 1, time.Time{}, false},
		{"999999 days", 999999 * 24, time.Time{}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, ok := maxAgeCutoff(now, tc.hours)

			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (ts %d)", ok, tc.wantOK, ts)
			}
			if !ok {
				return
			}
			if ts != tc.want.UnixMilli() {
				t.Errorf("ts = %d, want %d", ts, tc.want.UnixMilli())
			}
			if ts >= now.UnixMilli() {
				t.Errorf("cutoff %s is not in the past", time.UnixMilli(ts).UTC())
			}
		})
	}
}
