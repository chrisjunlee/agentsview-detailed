package db

import (
	"testing"
	"time"
)

// TestUsageFilterFromTimestamp asserts the exact UTC timestamp
// strings produced by UsageFilter.FromTimestamp for both the
// no-FromTime (default midnight) and explicit-FromTime paths.
func TestUsageFilterFromTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		fromTime string
		want     string
	}{
		{
			name:     "no FromTime defaults to midnight",
			from:     "2026-05-30",
			fromTime: "",
			want:     "2026-05-30T00:00:00Z",
		},
		{
			name:     "explicit FromTime appends :00 seconds",
			from:     "2026-05-30",
			fromTime: "09:15",
			want:     "2026-05-30T09:15:00Z",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := UsageFilter{From: tt.from, FromTime: tt.fromTime}
			if got := f.FromTimestamp(); got != tt.want {
				t.Errorf("FromTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUsageFilterToTimestamp asserts the exact UTC timestamp
// strings produced by UsageFilter.ToTimestamp for both the
// no-ToTime (end-of-day) and explicit-ToTime paths.
// Note: an explicit ToTime gets :59 seconds appended, NOT :00.
func TestUsageFilterToTimestamp(t *testing.T) {
	tests := []struct {
		name   string
		to     string
		toTime string
		want   string
	}{
		{
			name:   "no ToTime defaults to end of day",
			to:     "2026-05-30",
			toTime: "",
			want:   "2026-05-30T23:59:59Z",
		},
		{
			name:   "explicit ToTime appends :59 seconds",
			to:     "2026-05-30",
			toTime: "14:30",
			want:   "2026-05-30T14:30:59Z",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := UsageFilter{To: tt.to, ToTime: tt.toTime}
			if got := f.ToTimestamp(); got != tt.want {
				t.Errorf("ToTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestUsageFilterInLocalRange exercises the boundary logic of
// UsageFilter.inLocalRange using time.UTC as the location so
// the test is timezone-independent.
func TestUsageFilterInLocalRange(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name        string
		filter      UsageFilter
		ts          string
		wantDate    string
		wantInRange bool
	}{
		// --- Date-range checks (no time bounds) ---
		{
			name:        "ts inside date range with no time bounds",
			filter:      UsageFilter{From: "2026-05-30", To: "2026-05-30"},
			ts:          "2026-05-30T09:15:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},
		{
			name:        "ts on From date matches",
			filter:      UsageFilter{From: "2026-05-29", To: "2026-05-31"},
			ts:          "2026-05-29T00:00:00Z",
			wantDate:    "2026-05-29",
			wantInRange: true,
		},
		{
			name:        "ts on To date matches",
			filter:      UsageFilter{From: "2026-05-29", To: "2026-05-31"},
			ts:          "2026-05-31T23:59:59Z",
			wantDate:    "2026-05-31",
			wantInRange: true,
		},
		{
			name:        "ts date before From is excluded",
			filter:      UsageFilter{From: "2026-05-30", To: "2026-05-31"},
			ts:          "2026-05-29T12:00:00Z",
			wantDate:    "2026-05-29",
			wantInRange: false,
		},
		{
			name:        "ts date after To is excluded",
			filter:      UsageFilter{From: "2026-05-29", To: "2026-05-30"},
			ts:          "2026-05-31T12:00:00Z",
			wantDate:    "2026-05-31",
			wantInRange: false,
		},

		// --- FromTime boundary (same-day) ---
		{
			name: "clock exactly equals FromTime is inclusive",
			filter: UsageFilter{
				From:     "2026-05-30",
				To:       "2026-05-30",
				FromTime: "09:15",
			},
			ts:          "2026-05-30T09:15:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},
		{
			name: "clock one minute before FromTime is excluded",
			filter: UsageFilter{
				From:     "2026-05-30",
				To:       "2026-05-30",
				FromTime: "09:15",
			},
			ts:          "2026-05-30T09:14:00Z",
			wantDate:    "2026-05-30",
			wantInRange: false,
		},
		{
			name: "clock after FromTime is included",
			filter: UsageFilter{
				From:     "2026-05-30",
				To:       "2026-05-30",
				FromTime: "09:15",
			},
			ts:          "2026-05-30T09:16:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},

		// --- ToTime boundary (same-day) ---
		{
			name: "clock exactly equals ToTime is inclusive",
			filter: UsageFilter{
				From:   "2026-05-30",
				To:     "2026-05-30",
				ToTime: "14:30",
			},
			ts:          "2026-05-30T14:30:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},
		{
			name: "clock one minute after ToTime is excluded",
			filter: UsageFilter{
				From:   "2026-05-30",
				To:     "2026-05-30",
				ToTime: "14:30",
			},
			ts:          "2026-05-30T14:31:00Z",
			wantDate:    "2026-05-30",
			wantInRange: false,
		},
		{
			name: "clock before ToTime is included",
			filter: UsageFilter{
				From:   "2026-05-30",
				To:     "2026-05-30",
				ToTime: "14:30",
			},
			ts:          "2026-05-30T14:29:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},

		// --- FromTime does not restrict a different date in range ---
		{
			name: "FromTime restriction only applies on From date",
			filter: UsageFilter{
				From:     "2026-05-30",
				To:       "2026-05-31",
				FromTime: "09:15",
			},
			// The next day at 08:00 is before FromTime but NOT on
			// the From date, so it should pass the time check.
			ts:          "2026-05-31T08:00:00Z",
			wantDate:    "2026-05-31",
			wantInRange: true,
		},

		// --- Unparseable timestamp fallback ---
		{
			name:        "garbage ts falls back to date-only slice",
			filter:      UsageFilter{From: "2026-05-30", To: "2026-05-30"},
			ts:          "garbage-string",
			wantDate:    "garbage-st",
			wantInRange: false,
		},
		{
			name:   "ts with correct date prefix but invalid time falls back",
			filter: UsageFilter{From: "2026-05-30", To: "2026-05-30"},
			// Not valid RFC3339 but has a 10-char date prefix.
			ts:          "2026-05-30Tbogus",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},
		{
			name:        "ts shorter than 10 chars returns empty date",
			filter:      UsageFilter{From: "2026-05-30", To: "2026-05-30"},
			ts:          "short",
			wantDate:    "",
			wantInRange: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotOK := tt.filter.inLocalRange(tt.ts, loc)
			if gotDate != tt.wantDate {
				t.Errorf("inLocalRange() date = %q, want %q",
					gotDate, tt.wantDate)
			}
			if gotOK != tt.wantInRange {
				t.Errorf("inLocalRange() ok = %v, want %v",
					gotOK, tt.wantInRange)
			}
		})
	}
}
