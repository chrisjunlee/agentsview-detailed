package pricing

// ModelAliases maps model IDs as they appear in Pi session
// logs to their corresponding LiteLLM pricing keys. This
// allows cost estimation for models accessed via subscription
// APIs (opencode.ai, OpenRouter free tier) that report zero
// cost in session data.
var ModelAliases = map[string]string{
	// opencode-go provider models (bare names)
	"deepseek-v4-flash": "deepseek/deepseek-v3.2",
	"deepseek-v4-pro":   "deepseek/deepseek-v3",
	"kimi-k2.6":         "moonshot/kimi-k2.6",
	"kimi-k2.5":         "moonshot/kimi-k2.5",
	"glm-5":             "zai/glm-5",
	"glm-5.1":           "zai/glm-5",
	"minimax-m2.5":      "minimax/MiniMax-M2.5",
	"minimax-m2.7":      "minimax/MiniMax-M2.5",
	"qwen3.5-plus":      "openrouter/qwen/qwen3.5-plus-02-15",
	"qwen3.6-plus":      "openrouter/qwen/qwen3.5-plus-02-15",
	"mimo-v2.5":         "novita/xiaomimimo/mimo-v2-flash",
	"mimo-v2-omni":      "novita/xiaomimimo/mimo-v2-flash",

	// opencode-go provider models (prefixed variants seen in session logs)
	"opencode-go/deepseek-v4-flash": "deepseek/deepseek-v3.2",
	"opencode-go/deepseek-v4-pro":   "deepseek/deepseek-v3",
	"opencode-go/kimi-k2.6":         "moonshot/kimi-k2.6",
	"opencode-go/kimi-k2.5":         "moonshot/kimi-k2.5",
	"opencode-go/glm-5":             "zai/glm-5",
	"opencode-go/glm-5.1":           "zai/glm-5",
	"opencode-go/minimax-m2.5":      "minimax/MiniMax-M2.5",
	"opencode-go/minimax-m2.7":      "minimax/MiniMax-M2.5",
	"opencode-go/qwen3.5-plus":      "openrouter/qwen/qwen3.5-plus-02-15",
	"opencode-go/qwen3.6-plus":      "openrouter/qwen/qwen3.5-plus-02-15",
	"opencode-go/mimo-v2.5":         "novita/xiaomimimo/mimo-v2-flash",
	"opencode-go/mimo-v2.5-pro":     "mimo-v2.5-pro",
	"opencode-go/mimo-v2-omni":      "novita/xiaomimimo/mimo-v2-flash",
	"opencode-go/mimo-v2-pro":       "mimo-v2-pro",

	// Moonshot-prefixed variants
	"moonshotai/kimi-k2.6": "moonshot/kimi-k2.6",
	"moonshotai/kimi-k2.5": "moonshot/kimi-k2.5",

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
