package pricing

import "testing"

// TestModelAliases_ExactMapping pins every entry in ModelAliases to its exact
// target. alias_integrity_test.go proves structural properties (no empty
// target, no self-map, idempotent, target is priced), but nothing pins the
// concrete src->dst pairs. This table is a deliberate rebase tripwire: the
// upcoming upstream rebase collides on Pi pricing, and a silent change to any
// alias mapping would re-route a Pi model's cost to the wrong rate without
// failing any other test.
//
// If you intentionally add, remove, or repoint an alias, update this table in
// the SAME commit. A diff here is the signal to double-check the mapping
// against ~/projects/api-cost-comparison/pricing.json.
func TestModelAliases_ExactMapping(t *testing.T) {
	want := map[string]string{
		// Bare names that also appear under LiteLLM keys.
		"kimi-k2.5": "moonshot/kimi-k2.5",
		"glm-5":     "zai/glm-5",

		// LiteLLM entries with wrong rates -> bare fallback name.
		"qwen3.6-plus": "qwen3.5-plus",
		"mimo-v2-omni": "mimo-v2.5",
		"minimax-m2.5": "minimax/MiniMax-M2.5",

		// Prefixed variants -> bare session names.
		"deepseek/deepseek-v4-flash":    "deepseek-v4-flash",
		"opencode-go/deepseek-v4-flash": "deepseek-v4-flash",
		"opencode-go/deepseek-v4-pro":   "deepseek-v4-pro",
		"opencode-go/kimi-k2.6":         "kimi-k2.6",
		"opencode-go/kimi-k2.5":         "moonshot/kimi-k2.5",
		"opencode-go/glm-5":             "glm-5.1",
		"opencode-go/glm-5.1":           "glm-5.1",
		"opencode-go/minimax-m2.5":      "minimax/MiniMax-M2.5",
		"opencode-go/minimax-m2.7":      "minimax-m2.7",
		"opencode-go/qwen3.5-plus":      "qwen3.5-plus",
		"opencode-go/qwen3.6-plus":      "qwen3.5-plus",
		"opencode-go/mimo-v2.5":         "mimo-v2.5",
		"opencode-go/mimo-v2.5-pro":     "mimo-v2.5-pro",
		"opencode-go/mimo-v2-omni":      "mimo-v2.5",
		"opencode-go/mimo-v2-pro":       "mimo-v2-pro",

		// Moonshot-prefixed variants.
		"moonshotai/kimi-k2.6": "kimi-k2.6",
		"moonshotai/kimi-k2.5": "moonshot/kimi-k2.5",

		// Free-tier names seen in session data.
		"minimax-m2.5-free":     "minimax/MiniMax-M2.5",
		"nemotron-3-super-free": "nvidia.nemotron-super-3-120b",

		// OpenRouter free-tier -> paid equivalents.
		"qwen/qwen3-coder:free":                     "openrouter/qwen/qwen3-coder",
		"openai/gpt-oss-120b:free":                  "deepinfra/openai/gpt-oss-120b",
		"nvidia/nemotron-3-super-120b-a12b:free":    "nvidia.nemotron-super-3-120b",
		"meta-llama/llama-3.3-70b-instruct:free":    "deepinfra/meta-llama/Llama-3.3-70B-Instruct",
		"nousresearch/hermes-3-llama-3.1-405b:free": "deepinfra/NousResearch/Hermes-3-Llama-3.1-405B",
		"google/gemma-4-31b-it:free":                "google/gemma-4-31b-it",
	}

	if len(ModelAliases) != len(want) {
		t.Errorf(
			"ModelAliases has %d entries, table pins %d; update this table "+
				"to match aliases.go in the same commit",
			len(ModelAliases), len(want),
		)
	}

	// Every pinned mapping must resolve exactly.
	for src, dst := range want {
		if got := ResolveModelAlias(src); got != dst {
			t.Errorf("ResolveModelAlias(%q) = %q, want %q", src, got, dst)
		}
	}

	// Every live alias must be present in the table (catches new entries
	// added during a rebase without a corresponding test update).
	for src, dst := range ModelAliases {
		if _, ok := want[src]; !ok {
			t.Errorf(
				"ModelAliases[%q] = %q is not pinned in this table; add it",
				src, dst,
			)
		}
	}
}
