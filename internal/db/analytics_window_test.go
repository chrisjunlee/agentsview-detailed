package db

import (
	"context"
	"testing"
	"time"
)

// TestAnalyticsFilterUtcRange_BothEmpty asserts that utcRange returns
// the unbounded sentinels when neither From nor To is set.
func TestAnalyticsFilterUtcRange_BothEmpty(t *testing.T) {
	f := AnalyticsFilter{}
	gotFrom, gotTo := f.utcRange()
	// With no dates the ±14h padding is skipped, so the sentinels
	// pass through unchanged.
	if gotFrom != "0001-01-01T00:00:00Z" {
		t.Errorf("utcRange from = %q, want %q",
			gotFrom, "0001-01-01T00:00:00Z")
	}
	if gotTo != "9999-12-31T23:59:59Z" {
		t.Errorf("utcRange to = %q, want %q",
			gotTo, "9999-12-31T23:59:59Z")
	}
}

// TestAnalyticsFilterUtcRange_FromTimeOnly asserts that when only
// FromTime is set the From bound uses "HH:MM:00Z" and the To bound
// uses "23:59:59Z" default.
func TestAnalyticsFilterUtcRange_FromTimeOnly(t *testing.T) {
	f := AnalyticsFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		FromTime: "09:15",
	}
	gotFrom, gotTo := f.utcRange()

	// The raw bound before padding is "2026-05-30T09:15:00Z".
	// After -14h padding: "2026-05-29T19:15:00Z".
	wantFromRaw := "2026-05-29T19:15:00Z"
	// The raw to bound (no ToTime) is "2026-05-30T23:59:59Z".
	// After +14h padding: "2026-05-31T13:59:59Z".
	wantToRaw := "2026-05-31T13:59:59Z"
	if gotFrom != wantFromRaw {
		t.Errorf("utcRange from = %q, want %q", gotFrom, wantFromRaw)
	}
	if gotTo != wantToRaw {
		t.Errorf("utcRange to = %q, want %q", gotTo, wantToRaw)
	}
}

// TestAnalyticsFilterUtcRange_ToTimeOnly asserts that when only
// ToTime is set the To bound uses "HH:MM:59Z" (note :59 suffix) and
// the From bound uses "00:00:00Z" default.
func TestAnalyticsFilterUtcRange_ToTimeOnly(t *testing.T) {
	f := AnalyticsFilter{
		From:   "2026-05-30",
		To:     "2026-05-30",
		ToTime: "14:30",
	}
	gotFrom, gotTo := f.utcRange()

	// Raw from (no FromTime): "2026-05-30T00:00:00Z" - 14h = "2026-05-29T10:00:00Z".
	wantFromRaw := "2026-05-29T10:00:00Z"
	// Raw to (ToTime 14:30): "2026-05-30T14:30:59Z" + 14h = "2026-05-31T04:30:59Z".
	wantToRaw := "2026-05-31T04:30:59Z"
	if gotFrom != wantFromRaw {
		t.Errorf("utcRange from = %q, want %q", gotFrom, wantFromRaw)
	}
	if gotTo != wantToRaw {
		t.Errorf("utcRange to = %q, want %q", gotTo, wantToRaw)
	}
}

// TestAnalyticsFilterUtcRange_BothSet asserts that when both
// FromTime and ToTime are set the bounds use HH:MM:00Z and HH:MM:59Z
// respectively (the :59 suffix is only on ToTime).
func TestAnalyticsFilterUtcRange_BothSet(t *testing.T) {
	f := AnalyticsFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		FromTime: "09:15",
		ToTime:   "14:30",
	}
	gotFrom, gotTo := f.utcRange()

	// Raw from "2026-05-30T09:15:00Z" - 14h = "2026-05-29T19:15:00Z".
	wantFromRaw := "2026-05-29T19:15:00Z"
	// Raw to "2026-05-30T14:30:59Z" + 14h = "2026-05-31T04:30:59Z".
	wantToRaw := "2026-05-31T04:30:59Z"
	if gotFrom != wantFromRaw {
		t.Errorf("utcRange from = %q, want %q", gotFrom, wantFromRaw)
	}
	if gotTo != wantToRaw {
		t.Errorf("utcRange to = %q, want %q", gotTo, wantToRaw)
	}
}

// TestAnalyticsFilterInLocalRange_Boundaries tests that
// inLocalRange is inclusive at both the from-time and to-time
// boundaries (== passes).
func TestAnalyticsFilterInLocalRange_Boundaries(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name        string
		filter      AnalyticsFilter
		ts          string
		wantDate    string
		wantInRange bool
	}{
		// --- basic date range, no time bounds ---
		{
			name: "ts inside date range no time bounds",
			filter: AnalyticsFilter{
				From: "2026-05-30", To: "2026-05-30",
			},
			ts:          "2026-05-30T12:00:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},
		{
			name: "ts before From excluded",
			filter: AnalyticsFilter{
				From: "2026-05-30", To: "2026-05-31",
			},
			ts:          "2026-05-29T23:59:59Z",
			wantDate:    "2026-05-29",
			wantInRange: false,
		},
		{
			name: "ts after To excluded",
			filter: AnalyticsFilter{
				From: "2026-05-29", To: "2026-05-30",
			},
			ts:          "2026-05-31T00:00:00Z",
			wantDate:    "2026-05-31",
			wantInRange: false,
		},

		// --- FromTime: boundary inclusive (== passes) ---
		{
			name: "clock exactly equals FromTime is inclusive",
			filter: AnalyticsFilter{
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
			filter: AnalyticsFilter{
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
			filter: AnalyticsFilter{
				From:     "2026-05-30",
				To:       "2026-05-30",
				FromTime: "09:15",
			},
			ts:          "2026-05-30T09:16:00Z",
			wantDate:    "2026-05-30",
			wantInRange: true,
		},

		// --- ToTime: boundary inclusive (== passes) ---
		{
			name: "clock exactly equals ToTime is inclusive",
			filter: AnalyticsFilter{
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
			filter: AnalyticsFilter{
				From:   "2026-05-30",
				To:     "2026-05-30",
				ToTime: "14:30",
			},
			ts:          "2026-05-30T14:31:00Z",
			wantDate:    "2026-05-30",
			wantInRange: false,
		},

		// --- time restriction only applies on the boundary date ---
		{
			name: "FromTime restriction does not apply on inner date",
			filter: AnalyticsFilter{
				From:     "2026-05-30",
				To:       "2026-05-31",
				FromTime: "09:15",
			},
			// 08:00 is before FromTime but this is NOT the From date.
			ts:          "2026-05-31T08:00:00Z",
			wantDate:    "2026-05-31",
			wantInRange: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotOK := tt.filter.inLocalRange(tt.ts, loc)
			if gotDate != tt.wantDate {
				t.Errorf(
					"inLocalRange() date = %q, want %q",
					gotDate, tt.wantDate,
				)
			}
			if gotOK != tt.wantInRange {
				t.Errorf(
					"inLocalRange() ok = %v, want %v",
					gotOK, tt.wantInRange,
				)
			}
		})
	}
}

// TestAnalyticsFilterInLocalRange_UnparseableFallback asserts that an
// unparseable timestamp falls back to ts[:10] for the date, and that
// a ts shorter than 10 chars produces an empty date.
func TestAnalyticsFilterInLocalRange_UnparseableFallback(t *testing.T) {
	loc := time.UTC

	t.Run("garbage ts fallback to first 10 chars", func(t *testing.T) {
		f := AnalyticsFilter{From: "2026-05-30", To: "2026-05-30"}
		gotDate, gotOK := f.inLocalRange("garbage-string", loc)
		// ts[:10] = "garbage-st", which is < "2026-05-30", excluded.
		if gotDate != "garbage-st" {
			t.Errorf("inLocalRange() date = %q, want %q",
				gotDate, "garbage-st")
		}
		if gotOK {
			t.Error("inLocalRange() ok = true, want false for out-of-range date")
		}
	})

	t.Run("10-char date prefix with invalid tail matches date range", func(t *testing.T) {
		f := AnalyticsFilter{From: "2026-05-30", To: "2026-05-30"}
		// Not valid RFC3339 but has a 10-char date prefix that matches.
		gotDate, gotOK := f.inLocalRange("2026-05-30Tbogus", loc)
		if gotDate != "2026-05-30" {
			t.Errorf("inLocalRange() date = %q, want %q",
				gotDate, "2026-05-30")
		}
		if !gotOK {
			t.Error("inLocalRange() ok = false, want true (date is in range)")
		}
	})

	t.Run("ts shorter than 10 chars returns empty date", func(t *testing.T) {
		f := AnalyticsFilter{From: "2026-05-30", To: "2026-05-30"}
		gotDate, gotOK := f.inLocalRange("short", loc)
		if gotDate != "" {
			t.Errorf("inLocalRange() date = %q, want %q", gotDate, "")
		}
		if gotOK {
			t.Error("inLocalRange() ok = true, want false")
		}
	})
}

// TestGetAnalyticsSummary_TimeWindow seeds sessions spanning a day
// boundary and verifies that GetAnalyticsSummary respects the
// from_time/to_time window: sessions starting before the from_time
// on the From date must be excluded.
func TestGetAnalyticsSummary_TimeWindow(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	// Session A: starts at 08:00 UTC on 2026-05-30 — BEFORE the window.
	insertSession(t, d, "win-early", "proj", func(s *Session) {
		ts := "2026-05-30T08:00:00Z"
		s.StartedAt = &ts
		s.EndedAt = &ts
		s.MessageCount = 2
	})
	insertMessages(t, d,
		Message{
			SessionID: "win-early", Ordinal: 0, Role: "user",
			Content: "early msg", ContentLength: 9,
			Timestamp: "2026-05-30T08:00:00Z",
		},
		Message{
			SessionID: "win-early", Ordinal: 1, Role: "assistant",
			Content: "early reply", ContentLength: 11,
			Timestamp: "2026-05-30T08:01:00Z",
		},
	)

	// Session B: starts at 10:00 UTC on 2026-05-30 — INSIDE the window.
	insertSession(t, d, "win-inside", "proj", func(s *Session) {
		ts := "2026-05-30T10:00:00Z"
		s.StartedAt = &ts
		s.EndedAt = &ts
		s.MessageCount = 2
	})
	insertMessages(t, d,
		Message{
			SessionID: "win-inside", Ordinal: 0, Role: "user",
			Content: "inside msg", ContentLength: 10,
			Timestamp: "2026-05-30T10:00:00Z",
		},
		Message{
			SessionID: "win-inside", Ordinal: 1, Role: "assistant",
			Content: "inside reply", ContentLength: 12,
			Timestamp: "2026-05-30T10:01:00Z",
		},
	)

	f := AnalyticsFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		FromTime: "09:00", // window starts at 09:00
		Timezone: "UTC",
	}
	got, err := d.GetAnalyticsSummary(ctx, f)
	if err != nil {
		t.Fatalf("GetAnalyticsSummary: %v", err)
	}

	// Only the 10:00 session should be counted.
	if got.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", got.TotalSessions)
	}
	if got.TotalMessages != 2 {
		t.Errorf("TotalMessages = %d, want 2", got.TotalMessages)
	}
}

// TestGetAnalyticsSummary_ToTimeWindow verifies that sessions after
// the to_time boundary on the To date are excluded.
func TestGetAnalyticsSummary_ToTimeWindow(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	// Session A: starts at 12:00 — inside the window.
	insertSession(t, d, "towin-in", "proj", func(s *Session) {
		ts := "2026-05-30T12:00:00Z"
		s.StartedAt = &ts
		s.EndedAt = &ts
		s.MessageCount = 2
	})
	insertMessages(t, d,
		Message{
			SessionID: "towin-in", Ordinal: 0, Role: "user",
			Content: "in msg", ContentLength: 6,
			Timestamp: "2026-05-30T12:00:00Z",
		},
		Message{
			SessionID: "towin-in", Ordinal: 1, Role: "assistant",
			Content: "in reply", ContentLength: 8,
			Timestamp: "2026-05-30T12:01:00Z",
		},
	)

	// Session B: starts at 16:00 — after the to_time of 15:00.
	insertSession(t, d, "towin-out", "proj", func(s *Session) {
		ts := "2026-05-30T16:00:00Z"
		s.StartedAt = &ts
		s.EndedAt = &ts
		s.MessageCount = 2
	})
	insertMessages(t, d,
		Message{
			SessionID: "towin-out", Ordinal: 0, Role: "user",
			Content: "late msg", ContentLength: 8,
			Timestamp: "2026-05-30T16:00:00Z",
		},
		Message{
			SessionID: "towin-out", Ordinal: 1, Role: "assistant",
			Content: "late reply", ContentLength: 10,
			Timestamp: "2026-05-30T16:01:00Z",
		},
	)

	f := AnalyticsFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		ToTime:   "15:00",
		Timezone: "UTC",
	}
	got, err := d.GetAnalyticsSummary(ctx, f)
	if err != nil {
		t.Fatalf("GetAnalyticsSummary: %v", err)
	}

	if got.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", got.TotalSessions)
	}
}
