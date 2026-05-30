package db

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/wesm/agentsview/internal/pricing"
)

// TestPricingAliasResolution_DeepseekV4Pro verifies the end-to-end alias
// resolution path in loadPricingMap. The alias "opencode-go/deepseek-v4-pro"
// maps to "deepseek-v4-pro" in ModelAliases. When a message is inserted with
// the aliased model name, GetDailyUsage should compute a non-zero cost using
// the "deepseek-v4-pro" rates — proving loadPricingMap copies the target's
// rates onto the source key.
//
// Alias: opencode-go/deepseek-v4-pro -> deepseek-v4-pro
// Rates (from api-cost-comparison/pricing.json oracle):
//
//	input=1.74, output=3.48, cacheCreation=0, cacheRead=0.0145
func TestPricingAliasResolution_DeepseekV4Pro(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()

	// Seed fallback pricing — this is what loadPricingMap reads from
	// model_pricing. This mirrors the opus-4-8 test in usage_test.go.
	upsertFallbackPricing(t, d)

	// Verify the alias exists in ModelAliases to make the dependency explicit.
	aliasSrc := "opencode-go/deepseek-v4-pro"
	aliasDst := "deepseek-v4-pro"
	if got := pricing.ResolveModelAlias(aliasSrc); got != aliasDst {
		t.Fatalf("alias precondition failed: ResolveModelAlias(%q) = %q, want %q",
			aliasSrc, got, aliasDst)
	}

	// Insert a session and a message using the alias (src) model name.
	started := "2026-05-30T10:00:00Z"
	ended := "2026-05-30T11:00:00Z"
	insertSession(t, d, "alias-deepseek-pro", "proj-pi", func(s *Session) {
		s.Agent = "opencode"
		s.StartedAt = &started
		s.EndedAt = &ended
	})

	// Token counts chosen to give a clean, verifiable expected cost:
	//   inputTok=1_000_000, outputTok=500_000, cacheCr=0, cacheRd=200_000
	//
	// Expected cost using deepseek-v4-pro rates:
	//   (1_000_000 * 1.74 + 500_000 * 3.48 + 0 * 0 + 200_000 * 0.0145)
	//   / 1_000_000
	//   = (1_740_000 + 1_740_000 + 0 + 2_900) / 1_000_000
	//   = 3_482_900 / 1_000_000
	//   = 3.4829
	const (
		inputTok   = 1_000_000
		outputTok  = 500_000
		cacheRdTok = 200_000
	)

	tokenUsage := json.RawMessage(`{
		"input_tokens": 1000000,
		"output_tokens": 500000,
		"cache_creation_input_tokens": 0,
		"cache_read_input_tokens": 200000
	}`)
	insertMessages(t, d, Message{
		SessionID:  "alias-deepseek-pro",
		Ordinal:    0,
		Role:       "assistant",
		Timestamp:  "2026-05-30T10:30:00Z",
		Model:      aliasSrc, // the alias key, not the pricing key
		TokenUsage: tokenUsage,
	})

	result, err := d.GetDailyUsage(ctx, UsageFilter{
		From:     "2026-05-30",
		To:       "2026-05-30",
		Timezone: "UTC",
	})
	requireNoError(t, err, "GetDailyUsage with alias model")

	if len(result.Daily) != 1 {
		t.Fatalf("got %d daily entries, want 1", len(result.Daily))
	}

	day := result.Daily[0]
	if day.Date != "2026-05-30" {
		t.Errorf("Date = %q, want 2026-05-30", day.Date)
	}
	if day.InputTokens != inputTok {
		t.Errorf("InputTokens = %d, want %d", day.InputTokens, inputTok)
	}
	if day.OutputTokens != outputTok {
		t.Errorf("OutputTokens = %d, want %d", day.OutputTokens, outputTok)
	}
	if day.CacheReadTokens != cacheRdTok {
		t.Errorf("CacheReadTokens = %d, want %d", day.CacheReadTokens, cacheRdTok)
	}

	// deepseek-v4-pro rates: input=1.74, output=3.48, cacheRead=0.0145
	const (
		inputRate   = 1.74
		outputRate  = 3.48
		cacheRdRate = 0.0145
	)
	wantCost := (float64(inputTok)*inputRate +
		float64(outputTok)*outputRate +
		float64(cacheRdTok)*cacheRdRate) / 1_000_000
	// wantCost = (1_740_000 + 1_740_000 + 2_900) / 1_000_000 = 3.4829

	if day.TotalCost == 0 {
		t.Fatal("TotalCost = 0; alias was not resolved to deepseek-v4-pro rates")
	}
	if math.Abs(day.TotalCost-wantCost) > 1e-9 {
		t.Errorf("TotalCost = %v, want %v (alias %q -> %q not applied)",
			day.TotalCost, wantCost, aliasSrc, aliasDst)
	}
	if math.Abs(result.Totals.TotalCost-wantCost) > 1e-9 {
		t.Errorf("Totals.TotalCost = %v, want %v",
			result.Totals.TotalCost, wantCost)
	}
}
