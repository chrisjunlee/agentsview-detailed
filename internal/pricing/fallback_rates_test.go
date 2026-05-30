package pricing

import "testing"

// TestFallbackPricing_PiAndClaudeRates asserts exact ModelPricing values for
// each Pi model and Claude model row added in this fork. Values are taken from
// the oracle: ~/projects/api-cost-comparison/pricing.json.
//
// Do NOT add claude-opus-4-6, claude-opus-4-8, or FallbackVersion here —
// those are already covered by fallback_test.go.
func TestFallbackPricing_PiAndClaudeRates(t *testing.T) {
	type wantRow struct {
		pattern       string
		input         float64
		output        float64
		cacheCreation float64
		cacheRead     float64
		// noOracle marks rows with no entry in pricing.json;
		// characterization only — see comment per row below.
		noOracle bool
	}

	rows := []wantRow{
		// --- Pi models — rates from api-cost-comparison/pricing.json ---
		{
			pattern: "deepseek-v4-flash",
			input:   0.14, output: 0.28,
			cacheCreation: 0, cacheRead: 0.0028,
		},
		{
			pattern: "deepseek-v4-pro",
			input:   1.74, output: 3.48,
			cacheCreation: 0, cacheRead: 0.0145,
		},
		{
			pattern: "kimi-k2.6",
			input:   0.95, output: 4.00,
			cacheCreation: 0, cacheRead: 0.16,
		},
		{
			pattern: "glm-5.1",
			input:   1.40, output: 4.40,
			cacheCreation: 0, cacheRead: 0.26,
		},
		{
			pattern: "mimo-v2.5-pro",
			input:   1.00, output: 3.00,
			cacheCreation: 0, cacheRead: 0.20,
		},
		{
			pattern: "mimo-v2.5",
			input:   0.40, output: 2.00,
			cacheCreation: 0, cacheRead: 0.08,
		},
		{
			// no oracle entry in pricing.json; characterization only
			pattern: "mimo-v2-pro",
			input:   1.00, output: 3.00,
			cacheCreation: 0, cacheRead: 0,
			noOracle: true,
		},
		{
			pattern: "minimax-m2.7",
			input:   0.30, output: 1.20,
			cacheCreation: 0.375, cacheRead: 0.06,
		},
		{
			pattern: "qwen3.5-plus",
			input:   0.40, output: 2.40,
			cacheCreation: 0, cacheRead: 0.04,
		},
		{
			// no oracle entry in pricing.json; characterization only
			pattern: "google/gemma-4-31b-it",
			input:   0.10, output: 0.40,
			cacheCreation: 0, cacheRead: 0,
			noOracle: true,
		},
		// --- Claude models added in this fork ---
		{
			pattern: "claude-sonnet-4-6",
			input:   3.0, output: 15.0,
			cacheCreation: 3.75, cacheRead: 0.30,
		},
		{
			pattern: "claude-haiku-4-5-20251001",
			input:   1.0, output: 5.0,
			cacheCreation: 1.25, cacheRead: 0.10,
		},
	}

	prices := FallbackPricing()
	// Build a lookup map for O(1) access.
	byPattern := make(map[string]ModelPricing, len(prices))
	for _, p := range prices {
		byPattern[p.ModelPattern] = p
	}

	for _, w := range rows {
		t.Run(w.pattern, func(t *testing.T) {
			got, ok := byPattern[w.pattern]
			if !ok {
				t.Fatalf("pattern %q missing from FallbackPricing()", w.pattern)
			}
			want := ModelPricing{
				ModelPattern:         w.pattern,
				InputPerMTok:         w.input,
				OutputPerMTok:        w.output,
				CacheCreationPerMTok: w.cacheCreation,
				CacheReadPerMTok:     w.cacheRead,
			}
			if got != want {
				if w.noOracle {
					t.Errorf(
						"[characterization-only, no pricing.json oracle] "+
							"%q: got %+v, want %+v",
						w.pattern, got, want,
					)
				} else {
					// PRICING DIVERGENCE: the code does not match the
					// api-cost-comparison/pricing.json oracle. This is a
					// user-visible cost number — do not silence this failure.
					t.Errorf(
						"PRICING DIVERGENCE from oracle: "+
							"%q: got %+v, want %+v",
						w.pattern, got, want,
					)
				}
			}
		})
	}
}
