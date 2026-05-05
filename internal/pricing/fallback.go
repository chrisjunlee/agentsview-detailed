package pricing

// FallbackVersion must be bumped whenever FallbackPricing
// rates change so the startup seeder knows to re-upsert.
const FallbackVersion = "2026-05-03.1"

// FallbackPricing returns hardcoded pricing for key Claude
// models and Pi session models. Used when the LiteLLM fetch
// fails, and to override LiteLLM entries that have wrong rates.
// Prices in USD per million tokens.
func FallbackPricing() []ModelPricing {
	return []ModelPricing{
		// Current model names (used by Claude Code / Codex)
		{
			ModelPattern:         "claude-sonnet-4-6",
			InputPerMTok:         3.0,
			OutputPerMTok:        15.0,
			CacheCreationPerMTok: 3.75,
			CacheReadPerMTok:     0.30,
		},
		{
			ModelPattern:         "claude-opus-4-6",
			InputPerMTok:         5.0,
			OutputPerMTok:        25.0,
			CacheCreationPerMTok: 6.25,
			CacheReadPerMTok:     0.50,
		},
		{
			ModelPattern:         "claude-haiku-4-5-20251001",
			InputPerMTok:         1.0,
			OutputPerMTok:        5.0,
			CacheCreationPerMTok: 1.25,
			CacheReadPerMTok:     0.10,
		},
		// Codex / OpenAI models
		{
			ModelPattern:  "gpt-5.4",
			InputPerMTok:  2.50,
			OutputPerMTok: 15.0,
		},
		{
			ModelPattern:  "gpt-5.2-codex",
			InputPerMTok:  1.75,
			OutputPerMTok: 14.0,
		},
		{
			ModelPattern:  "gpt-5.3-codex",
			InputPerMTok:  1.75,
			OutputPerMTok: 14.0,
		},
		{
			ModelPattern:  "gpt-5.4-mini",
			InputPerMTok:  0.75,
			OutputPerMTok: 4.50,
		},
		{
			ModelPattern:  "gpt-5.4-nano",
			InputPerMTok:  0.20,
			OutputPerMTok: 1.25,
		},
		{
			ModelPattern:  "gpt-5.1-codex-max",
			InputPerMTok:  1.25,
			OutputPerMTok: 10.0,
		},
		// Older model names (still in some session logs)
		{
			ModelPattern:         "claude-sonnet-4-20250514",
			InputPerMTok:         3.0,
			OutputPerMTok:        15.0,
			CacheCreationPerMTok: 3.75,
			CacheReadPerMTok:     0.30,
		},
		{
			ModelPattern:         "claude-sonnet-4-5-20250514",
			InputPerMTok:         3.0,
			OutputPerMTok:        15.0,
			CacheCreationPerMTok: 3.75,
			CacheReadPerMTok:     0.30,
		},
		{
			ModelPattern:         "claude-opus-4-20250514",
			InputPerMTok:         15.0,
			OutputPerMTok:        75.0,
			CacheCreationPerMTok: 18.75,
			CacheReadPerMTok:     1.50,
		},
		{
			ModelPattern:         "claude-haiku-3-5-20241022",
			InputPerMTok:         0.80,
			OutputPerMTok:        4.0,
			CacheCreationPerMTok: 1.0,
			CacheReadPerMTok:     0.08,
		},
		// Pi models — rates from api-cost-comparison/pricing.json,
		// sourced from each provider's official pricing page.
		// Keyed by actual session model name so they take
		// precedence over (often wrong) LiteLLM alias lookups.
		{
			ModelPattern:     "deepseek-v4-flash",
			InputPerMTok:     0.14,
			OutputPerMTok:    0.28,
			CacheReadPerMTok: 0.0028,
		},
		{
			ModelPattern:     "deepseek-v4-pro",
			InputPerMTok:     1.74,
			OutputPerMTok:    3.48,
			CacheReadPerMTok: 0.0145,
		},
		{
			ModelPattern:     "kimi-k2.6",
			InputPerMTok:     0.95,
			OutputPerMTok:    4.00,
			CacheReadPerMTok: 0.16,
		},
		{
			ModelPattern:     "glm-5.1",
			InputPerMTok:     1.40,
			OutputPerMTok:    4.40,
			CacheReadPerMTok: 0.26,
		},
		{
			ModelPattern:     "mimo-v2.5-pro",
			InputPerMTok:     1.00,
			OutputPerMTok:    3.00,
			CacheReadPerMTok: 0.20,
		},
		{
			ModelPattern:     "mimo-v2.5",
			InputPerMTok:     0.40,
			OutputPerMTok:    2.00,
			CacheReadPerMTok: 0.08,
		},
		{
			ModelPattern:  "mimo-v2-pro",
			InputPerMTok:  1.00,
			OutputPerMTok: 3.00,
		},
		{
			ModelPattern:         "minimax-m2.7",
			InputPerMTok:         0.30,
			OutputPerMTok:        1.20,
			CacheCreationPerMTok: 0.375,
			CacheReadPerMTok:     0.06,
		},
		{
			ModelPattern:     "qwen3.5-plus",
			InputPerMTok:     0.40,
			OutputPerMTok:    2.40,
			CacheReadPerMTok: 0.04,
		},
		{
			ModelPattern:  "google/gemma-4-31b-it",
			InputPerMTok:  0.10,
			OutputPerMTok: 0.40,
		},
	}
}
