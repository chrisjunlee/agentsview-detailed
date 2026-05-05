package pricing

// ModelAliases maps model IDs as they appear in Pi session
// logs to pricing keys. Models with direct entries in
// FallbackPricing are keyed by their bare session name;
// aliases for those models point to the bare name so
// prefixed variants inherit the correct fallback rates
// instead of (often wrong) LiteLLM entries.
var ModelAliases = map[string]string{
	// Models with correct rates in FallbackPricing — bare
	// name aliases kept for models that also appear under
	// LiteLLM keys (alias is skipped when the bare name
	// already has pricing, but listed for documentation).
	"kimi-k2.5": "moonshot/kimi-k2.5",
	"glm-5":     "zai/glm-5",

	// Models whose LiteLLM entries have wrong rates — alias
	// to bare name (which has correct fallback rates).
	"qwen3.6-plus":  "qwen3.5-plus",
	"mimo-v2-omni":  "mimo-v2.5",
	"minimax-m2.5":  "minimax/MiniMax-M2.5",

	// Prefixed variants → bare session names
	"deepseek/deepseek-v4-flash":   "deepseek-v4-flash",
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

	// Moonshot-prefixed variants
	"moonshotai/kimi-k2.6": "kimi-k2.6",
	"moonshotai/kimi-k2.5": "moonshot/kimi-k2.5",

	// Free-tier model names seen in session data
	"minimax-m2.5-free":    "minimax/MiniMax-M2.5",
	"nemotron-3-super-free": "nvidia.nemotron-super-3-120b",

	// OpenRouter free-tier models → paid equivalents
	"qwen/qwen3-coder:free":                       "openrouter/qwen/qwen3-coder",
	"openai/gpt-oss-120b:free":                    "deepinfra/openai/gpt-oss-120b",
	"nvidia/nemotron-3-super-120b-a12b:free":      "nvidia.nemotron-super-3-120b",
	"meta-llama/llama-3.3-70b-instruct:free":      "deepinfra/meta-llama/Llama-3.3-70B-Instruct",
	"nousresearch/hermes-3-llama-3.1-405b:free":   "deepinfra/NousResearch/Hermes-3-Llama-3.1-405B",
	"google/gemma-4-31b-it:free":                  "google/gemma-4-31b-it",
}

// ResolveModelAlias returns the LiteLLM pricing key for a
// given model ID. If no alias exists, returns the original
// model ID unchanged.
func ResolveModelAlias(model string) string {
	if alias, ok := ModelAliases[model]; ok {
		return alias
	}
	return model
}
