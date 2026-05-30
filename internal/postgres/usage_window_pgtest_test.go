//go:build pgtest

package postgres

import (
	"context"
	"testing"

	"github.com/wesm/agentsview/internal/db"
)

// TestPGUsageFilterFromTimestamp characterizes the exact UTC timestamp
// strings produced by UsageFilter.FromTimestamp for both the
// no-FromTime (default midnight) and explicit-FromTime paths.
// This mirrors the SQLite characterization in
// internal/db/usage_window_test.go but runs against the same method
// used by the Postgres usage queries.
func TestPGUsageFilterFromTimestamp(t *testing.T) {
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
			f := db.UsageFilter{From: tt.from, FromTime: tt.fromTime}
			if got := f.FromTimestamp(); got != tt.want {
				t.Errorf("FromTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPGUsageFilterToTimestamp characterizes the exact UTC timestamp
// strings produced by UsageFilter.ToTimestamp.
// Crucially: an explicit ToTime gets :59 appended (not :00), so the
// entire HH:MM minute is included.
func TestPGUsageFilterToTimestamp(t *testing.T) {
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
			f := db.UsageFilter{To: tt.to, ToTime: tt.toTime}
			if got := f.ToTimestamp(); got != tt.want {
				t.Errorf("ToTimestamp() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestPGStoreGetDailyUsage_TimeWindow asserts the CORRECT minute-level
// from_time/to_time behaviour for the PG usage path (matching SQLite).
//
// KNOWN BUG (skipped): the PG GetDailyUsage/GetTopSessions/GetTrendUsage
// post-loop filter only checks date < f.From / date > f.To — it does NOT
// call filterInLocalRange, so it ignores the HH:MM from_time/to_time bounds
// that the SQLite path (f.inLocalRange) honors. PG-backed usage cost/token
// totals therefore wrongly include boundary-date messages outside the
// window.
//
// This is a user-visible cost-number change, so per the test-suite design
// decision we PAUSE rather than auto-fix. The test is written against the
// CORRECT expectation (2.0) and skipped; remove the t.Skip once
// internal/postgres/usage.go applies filterInLocalRange in its usage
// post-loop. The analogous SQLite test in internal/db/usage_window_test.go
// already asserts this correct minute-level exclusion and passes.
func TestPGStoreGetDailyUsage_TimeWindow(t *testing.T) {
	t.Skip("KNOWN BUG: PG usage path ignores from_time/to_time HH:MM bounds; un-skip when internal/postgres/usage.go applies filterInLocalRange")

	_, store := prepareUsageSchema(t, "agentsview_usage_window_test")

	ctx := context.Background()

	// Insert pricing for our test model.
	_, err := store.DB().ExecContext(ctx, `
		INSERT INTO model_pricing (
			model_pattern, input_per_mtok, output_per_mtok,
			cache_creation_per_mtok, cache_read_per_mtok, updated_at
		) VALUES ('usage-win-model', 1.0, 2.0, 0.0, 0.0, 'seed')`)
	if err != nil {
		t.Fatalf("insert pricing: %v", err)
	}

	// Session containing two messages — one early (08:00) and one
	// inside (10:00).
	_, err = store.DB().ExecContext(ctx, `
		INSERT INTO sessions (
			id, machine, project, agent, started_at,
			message_count, user_message_count
		) VALUES (
			'usage-win-sess', 'test-machine', 'proj', 'claude',
			'2026-05-30T08:00:00Z'::timestamptz, 2, 1
		)`)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// Early message at 08:00 (1 M input tokens).
	_, err = store.DB().ExecContext(ctx, `
		INSERT INTO messages (
			session_id, ordinal, role, content, timestamp,
			content_length, model, token_usage
		) VALUES (
			'usage-win-sess', 0, 'assistant', 'early',
			'2026-05-30T08:00:00Z'::timestamptz, 5,
			'usage-win-model',
			'{"input_tokens":1000000}'
		)`)
	if err != nil {
		t.Fatalf("insert early message: %v", err)
	}

	// Inside message at 10:00 (2 M input tokens).
	_, err = store.DB().ExecContext(ctx, `
		INSERT INTO messages (
			session_id, ordinal, role, content, timestamp,
			content_length, model, token_usage
		) VALUES (
			'usage-win-sess', 1, 'assistant', 'inside',
			'2026-05-30T10:00:00Z'::timestamptz, 6,
			'usage-win-model',
			'{"input_tokens":2000000}'
		)`)
	if err != nil {
		t.Fatalf("insert inside message: %v", err)
	}

	// Filter with from_time=09:00.
	// CORRECT expected value (SQLite): 2.0 (only 10:00 message).
	// CURRENT PG behaviour: 3.0 (both messages included — bug).
	f := db.UsageFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		FromTime: "09:00",
		Timezone: "UTC",
	}
	result, err := store.GetDailyUsage(ctx, f)
	if err != nil {
		t.Fatalf("GetDailyUsage: %v", err)
	}

	// CORRECT minute-level expectation (matches the SQLite path): with
	// from_time=09:00 the 08:00 message is excluded, so only the 10:00
	// message (2M input tokens) counts. TotalCost = 2M * 1.0 / 1M = 2.0.
	// This is the value the test asserts once the PG usage path calls
	// filterInLocalRange; until then the function is skipped above.
	if got := result.Totals.TotalCost; got != 2.0 {
		t.Errorf(
			"TotalCost = %.2f, want 2.0 (correct minute-level window: "+
				"only the 10:00 message counts once from_time=09:00 is honored)",
			got,
		)
	}
}
