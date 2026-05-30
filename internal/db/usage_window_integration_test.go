package db

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"testing"
)

// The pure UsageFilter helpers (FromTimestamp/ToTimestamp/inLocalRange) are
// characterized in usage_window_test.go. Those tests prove the helper is
// correct in isolation but NOT that the three usage query functions actually
// call it. These integration tests close that gap: they seed a real SQLite DB
// and assert that GetDailyUsage, GetTopSessionsByCost, and GetUsageSessionCounts
// each honor the minute-level FromTime/ToTime bounds end-to-end. A regression
// that dropped the post-query f.inLocalRange call from any of these functions
// would leave the helper tests green but break these.
//
// This is also the SQLite oracle referenced by the skipped Postgres test in
// internal/postgres/usage_window_pgtest_test.go: the PG usage path currently
// ignores HH:MM bounds (date-only filter), so the correct behavior these tests
// pin is the target the PG fix must match.

const usageWinModel = "usage-win-model"

// seedUsageWinPricing inserts a single priced model whose input rate is
// 1.0/MTok and whose every other rate is 0, so cost == input_tokens / 1e6.
// That makes the expected cost for an N-million-token message exactly N.
func seedUsageWinPricing(t *testing.T, d *DB) {
	t.Helper()
	err := d.UpsertModelPricing([]ModelPricing{{
		ModelPattern: usageWinModel,
		InputPerMTok: 1.0,
	}})
	requireNoError(t, err, "UpsertModelPricing usage-win")
}

// usageWinMsg builds an eligible assistant message carrying input_tokens only.
func usageWinMsg(sid string, ord int, ts string, inputTokens int) Message {
	return Message{
		SessionID: sid,
		Ordinal:   ord,
		Role:      "assistant",
		Timestamp: ts,
		Model:     usageWinModel,
		TokenUsage: json.RawMessage(
			`{"input_tokens":` +
				strconv.Itoa(inputTokens) + `}`,
		),
	}
}

// TestGetDailyUsageHonorsFromTime proves GetDailyUsage applies the FromTime
// HH:MM bound: with from_time=09:00 the 08:00 message is excluded, so only the
// 10:00 message's tokens count. The no-bound sub-case (cost 3.0) doubles as the
// mutation guard — if the inLocalRange call were removed, the windowed cost
// would equal the unwindowed cost and the 2.0 assertion would fail.
func TestGetDailyUsageHonorsFromTime(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	seedUsageWinPricing(t, d)

	started := "2026-05-30T08:00:00Z"
	insertSession(t, d, "daily-win", "proj", func(s *Session) {
		s.Agent = "claude"
		s.StartedAt = &started
	})
	insertMessages(t, d,
		usageWinMsg("daily-win", 0, "2026-05-30T08:00:00Z", 1_000_000),
		usageWinMsg("daily-win", 1, "2026-05-30T10:00:00Z", 2_000_000),
	)

	// Sanity / mutation guard: no time bound includes both messages.
	full, err := d.GetDailyUsage(ctx, UsageFilter{
		From: "2026-05-30", To: "2026-05-30", Timezone: "UTC",
	})
	requireNoError(t, err, "GetDailyUsage no-bound")
	if got := full.Totals.TotalCost; math.Abs(got-3.0) > 1e-9 {
		t.Fatalf("no-bound TotalCost = %v, want 3.0 (both messages)", got)
	}

	// from_time=09:00 excludes the 08:00 message; only the 10:00 message
	// (2M input tokens) counts. TotalCost = 2M * 1.0 / 1M = 2.0.
	windowed, err := d.GetDailyUsage(ctx, UsageFilter{
		From: "2026-05-30", To: "2026-05-30",
		FromTime: "09:00", Timezone: "UTC",
	})
	requireNoError(t, err, "GetDailyUsage from_time=09:00")
	if got := windowed.Totals.TotalCost; math.Abs(got-2.0) > 1e-9 {
		t.Errorf(
			"TotalCost = %v, want 2.0 (from_time=09:00 must exclude the "+
				"08:00 message)", got,
		)
	}
}

// TestGetTopSessionsByCostHonorsToTime proves GetTopSessionsByCost applies the
// ToTime HH:MM bound: with to_time=14:30 the 14:31 message is excluded while the
// 14:30 message (exact boundary, inclusive) is kept. Only the 14:30 message's
// 1M tokens / 1.0 cost should be attributed to the session.
func TestGetTopSessionsByCostHonorsToTime(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	seedUsageWinPricing(t, d)

	started := "2026-05-30T14:00:00Z"
	insertSession(t, d, "top-win", "proj", func(s *Session) {
		s.Agent = "claude"
		s.StartedAt = &started
	})
	insertMessages(t, d,
		usageWinMsg("top-win", 0, "2026-05-30T14:30:00Z", 1_000_000),
		usageWinMsg("top-win", 1, "2026-05-30T14:31:00Z", 5_000_000),
	)

	// Sanity / mutation guard: no time bound includes both (cost 6.0).
	full, err := d.GetTopSessionsByCost(ctx, UsageFilter{
		From: "2026-05-30", To: "2026-05-30", Timezone: "UTC",
	}, 10)
	requireNoError(t, err, "GetTopSessionsByCost no-bound")
	if len(full) != 1 {
		t.Fatalf("no-bound got %d sessions, want 1", len(full))
	}
	if got := full[0].Cost; math.Abs(got-6.0) > 1e-9 {
		t.Fatalf("no-bound Cost = %v, want 6.0 (both messages)", got)
	}

	// to_time=14:30 keeps the 14:30 message (inclusive boundary) and
	// excludes the 14:31 message. Cost = 1M * 1.0 / 1M = 1.0.
	windowed, err := d.GetTopSessionsByCost(ctx, UsageFilter{
		From: "2026-05-30", To: "2026-05-30",
		ToTime: "14:30", Timezone: "UTC",
	}, 10)
	requireNoError(t, err, "GetTopSessionsByCost to_time=14:30")
	if len(windowed) != 1 {
		t.Fatalf("got %d sessions, want 1", len(windowed))
	}
	if got := windowed[0].Cost; math.Abs(got-1.0) > 1e-9 {
		t.Errorf(
			"Cost = %v, want 1.0 (to_time=14:30 must exclude the 14:31 "+
				"message but keep the 14:30 boundary message)", got,
		)
	}
	if got := windowed[0].TotalTokens; got != 1_000_000 {
		t.Errorf("TotalTokens = %d, want 1000000", got)
	}
}

// TestGetUsageSessionCountsHonorsFromTime proves GetUsageSessionCounts applies
// the FromTime bound when deciding which sessions qualify. Session A's only
// message is at 08:00 (before the window) so it must NOT be counted; session B
// has a 10:00 message and must be counted. With from_time=09:00 the distinct
// session total is therefore 1, not 2.
func TestGetUsageSessionCountsHonorsFromTime(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	seedUsageWinPricing(t, d)

	startedA := "2026-05-30T08:00:00Z"
	insertSession(t, d, "count-early", "proj", func(s *Session) {
		s.Agent = "claude"
		s.StartedAt = &startedA
	})
	startedB := "2026-05-30T10:00:00Z"
	insertSession(t, d, "count-inside", "proj", func(s *Session) {
		s.Agent = "claude"
		s.StartedAt = &startedB
	})
	insertMessages(t, d,
		usageWinMsg("count-early", 0, "2026-05-30T08:00:00Z", 1_000_000),
		usageWinMsg("count-inside", 0, "2026-05-30T10:00:00Z", 1_000_000),
	)

	// Sanity / mutation guard: no time bound counts both sessions.
	full, err := d.GetUsageSessionCounts(ctx, UsageFilter{
		From: "2026-05-30", To: "2026-05-30", Timezone: "UTC",
	})
	requireNoError(t, err, "GetUsageSessionCounts no-bound")
	if full.Total != 2 {
		t.Fatalf("no-bound Total = %d, want 2 (both sessions)", full.Total)
	}

	// from_time=09:00 excludes the 08:00-only session.
	windowed, err := d.GetUsageSessionCounts(ctx, UsageFilter{
		From: "2026-05-30", To: "2026-05-30",
		FromTime: "09:00", Timezone: "UTC",
	})
	requireNoError(t, err, "GetUsageSessionCounts from_time=09:00")
	if windowed.Total != 1 {
		t.Errorf(
			"Total = %d, want 1 (from_time=09:00 must exclude the session "+
				"whose only message is at 08:00)", windowed.Total,
		)
	}
	if got := windowed.ByProject["proj"]; got != 1 {
		t.Errorf("ByProject[proj] = %d, want 1", got)
	}
}
