//go:build pgtest

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/wesm/agentsview/internal/db"
)

// TestPGAnalyticsUTCRange_FromTimeOnly characterizes that
// analyticsUTCRange builds the correct padded bounds when only
// FromTime is set. The from bound uses HH:MM:00Z; the to bound uses
// the 23:59:59Z default — both are then padded by ±14h.
func TestPGAnalyticsUTCRange_FromTimeOnly(t *testing.T) {
	f := db.AnalyticsFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		FromTime: "09:15",
	}
	gotFrom, gotTo := analyticsUTCRange(f)

	// Raw from "2026-05-30T09:15:00Z" - 14h = "2026-05-29T19:15:00Z".
	wantFrom := "2026-05-29T19:15:00Z"
	// Raw to "2026-05-30T23:59:59Z" + 14h = "2026-05-31T13:59:59Z".
	wantTo := "2026-05-31T13:59:59Z"

	if gotFrom != wantFrom {
		t.Errorf("analyticsUTCRange from = %q, want %q", gotFrom, wantFrom)
	}
	if gotTo != wantTo {
		t.Errorf("analyticsUTCRange to = %q, want %q", gotTo, wantTo)
	}
}

// TestPGAnalyticsUTCRange_ToTimeOnly characterizes the ":59" seconds
// suffix that is appended to an explicit ToTime (not ":00").
func TestPGAnalyticsUTCRange_ToTimeOnly(t *testing.T) {
	f := db.AnalyticsFilter{
		From:   "2026-05-30",
		To:     "2026-05-30",
		ToTime: "14:30",
	}
	gotFrom, gotTo := analyticsUTCRange(f)

	// Raw from "2026-05-30T00:00:00Z" - 14h = "2026-05-29T10:00:00Z".
	wantFrom := "2026-05-29T10:00:00Z"
	// Raw to "2026-05-30T14:30:59Z" + 14h = "2026-05-31T04:30:59Z".
	wantTo := "2026-05-31T04:30:59Z"

	if gotFrom != wantFrom {
		t.Errorf("analyticsUTCRange from = %q, want %q", gotFrom, wantFrom)
	}
	if gotTo != wantTo {
		t.Errorf("analyticsUTCRange to = %q, want %q", gotTo, wantTo)
	}
}

// TestPGAnalyticsUTCRange_BothEmpty characterizes that both bounds
// pass through unchanged as wide-open sentinels when neither From
// nor To is set.
func TestPGAnalyticsUTCRange_BothEmpty(t *testing.T) {
	f := db.AnalyticsFilter{}
	gotFrom, gotTo := analyticsUTCRange(f)
	if gotFrom != "0001-01-01T00:00:00Z" {
		t.Errorf("analyticsUTCRange from = %q, want %q",
			gotFrom, "0001-01-01T00:00:00Z")
	}
	if gotTo != "9999-12-31T23:59:59Z" {
		t.Errorf("analyticsUTCRange to = %q, want %q",
			gotTo, "9999-12-31T23:59:59Z")
	}
}

// TestPGFilterInLocalRange_Boundaries exercises filterInLocalRange
// (the package-level function mirroring db.AnalyticsFilter.inLocalRange).
// Boundaries must be inclusive; the fallback path for unparseable
// timestamps must use ts[:10].
func TestPGFilterInLocalRange_Boundaries(t *testing.T) {
	loc := time.UTC

	tests := []struct {
		name        string
		filter      db.AnalyticsFilter
		ts          string
		wantDate    string
		wantInRange bool
	}{
		{
			name:     "FromTime boundary inclusive",
			filter:   db.AnalyticsFilter{From: "2026-05-30", To: "2026-05-30", FromTime: "09:15"},
			ts:       "2026-05-30T09:15:00Z",
			wantDate: "2026-05-30", wantInRange: true,
		},
		{
			name:     "one minute before FromTime excluded",
			filter:   db.AnalyticsFilter{From: "2026-05-30", To: "2026-05-30", FromTime: "09:15"},
			ts:       "2026-05-30T09:14:00Z",
			wantDate: "2026-05-30", wantInRange: false,
		},
		{
			name:     "ToTime boundary inclusive",
			filter:   db.AnalyticsFilter{From: "2026-05-30", To: "2026-05-30", ToTime: "14:30"},
			ts:       "2026-05-30T14:30:00Z",
			wantDate: "2026-05-30", wantInRange: true,
		},
		{
			name:     "one minute after ToTime excluded",
			filter:   db.AnalyticsFilter{From: "2026-05-30", To: "2026-05-30", ToTime: "14:30"},
			ts:       "2026-05-30T14:31:00Z",
			wantDate: "2026-05-30", wantInRange: false,
		},
		{
			name:     "unparseable ts fallback to first 10 chars",
			filter:   db.AnalyticsFilter{From: "2026-05-30", To: "2026-05-30"},
			ts:       "garbage-string",
			wantDate: "garbage-st", wantInRange: false,
		},
		{
			name:     "10-char date prefix with invalid tail matches",
			filter:   db.AnalyticsFilter{From: "2026-05-30", To: "2026-05-30"},
			ts:       "2026-05-30Tbogus",
			wantDate: "2026-05-30", wantInRange: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			gotDate, gotOK := filterInLocalRange(tt.filter, tt.ts, loc)
			if gotDate != tt.wantDate {
				t.Errorf("filterInLocalRange() date = %q, want %q",
					gotDate, tt.wantDate)
			}
			if gotOK != tt.wantInRange {
				t.Errorf("filterInLocalRange() ok = %v, want %v",
					gotOK, tt.wantInRange)
			}
		})
	}
}

// TestPGStoreGetAnalyticsSummary_TimeWindow is an end-to-end pgtest
// that seeds sessions at different hours, pushes them to Postgres,
// then verifies that GetAnalyticsSummary on the Store respects the
// from_time window.
func TestPGStoreGetAnalyticsSummary_TimeWindow(t *testing.T) {
	pgURL := testPGURL(t)
	cleanPGSchema(t, pgURL)
	t.Cleanup(func() { cleanPGSchema(t, pgURL) })

	local := testDB(t)
	ps, err := New(
		pgURL, "agentsview", local,
		"pg-analytics-window-machine", true,
		SyncOptions{},
	)
	if err != nil {
		t.Fatalf("creating sync: %v", err)
	}
	defer ps.Close()

	ctx := context.Background()
	if err := ps.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// Session A: starts at 08:00 — before the from_time window.
	earlyStart := "2026-05-30T08:00:00Z"
	if err := local.UpsertSession(db.Session{
		ID:           "pg-aw-early",
		Project:      "proj",
		Machine:      "pg-analytics-window-machine",
		Agent:        "claude",
		MessageCount: 2,
		StartedAt:    &earlyStart,
	}); err != nil {
		t.Fatalf("upsert early session: %v", err)
	}
	if err := local.InsertMessages([]db.Message{
		{
			SessionID:     "pg-aw-early",
			Ordinal:       0,
			Role:          "user",
			Content:       "early",
			ContentLength: 5,
			Timestamp:     "2026-05-30T08:00:00Z",
		},
		{
			SessionID:     "pg-aw-early",
			Ordinal:       1,
			Role:          "assistant",
			Content:       "reply",
			ContentLength: 5,
			Timestamp:     "2026-05-30T08:01:00Z",
		},
	}); err != nil {
		t.Fatalf("insert early messages: %v", err)
	}

	// Session B: starts at 10:00 — inside the from_time window.
	insideStart := "2026-05-30T10:00:00Z"
	if err := local.UpsertSession(db.Session{
		ID:           "pg-aw-inside",
		Project:      "proj",
		Machine:      "pg-analytics-window-machine",
		Agent:        "claude",
		MessageCount: 2,
		StartedAt:    &insideStart,
	}); err != nil {
		t.Fatalf("upsert inside session: %v", err)
	}
	if err := local.InsertMessages([]db.Message{
		{
			SessionID:     "pg-aw-inside",
			Ordinal:       0,
			Role:          "user",
			Content:       "inside",
			ContentLength: 6,
			Timestamp:     "2026-05-30T10:00:00Z",
		},
		{
			SessionID:     "pg-aw-inside",
			Ordinal:       1,
			Role:          "assistant",
			Content:       "reply",
			ContentLength: 5,
			Timestamp:     "2026-05-30T10:01:00Z",
		},
	}); err != nil {
		t.Fatalf("insert inside messages: %v", err)
	}

	if _, err := ps.Push(ctx, false, nil); err != nil {
		t.Fatalf("push: %v", err)
	}

	store, err := NewStore(pgURL, "agentsview", true)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	f := db.AnalyticsFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		FromTime: "09:00",
		Timezone: "UTC",
	}
	got, err := store.GetAnalyticsSummary(ctx, f)
	if err != nil {
		t.Fatalf("GetAnalyticsSummary: %v", err)
	}

	// Only the 10:00 session should pass the from_time=09:00 filter.
	if got.TotalSessions != 1 {
		t.Errorf("TotalSessions = %d, want 1", got.TotalSessions)
	}
	if got.TotalMessages != 2 {
		t.Errorf("TotalMessages = %d, want 2", got.TotalMessages)
	}
}
