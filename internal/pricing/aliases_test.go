package pricing

import "testing"

func TestResolveModelAlias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"kimi-k2.5", "moonshot/kimi-k2.5"},
		{"qwen/qwen3-coder:free", "openrouter/qwen/qwen3-coder"},
		{"meta-llama/llama-3.3-70b-instruct:free", "deepinfra/meta-llama/Llama-3.3-70B-Instruct"},
		// Models with direct fallback entries — no alias
		{"deepseek-v4-flash", "deepseek-v4-flash"},
		{"deepseek-v4-pro", "deepseek-v4-pro"},
		{"kimi-k2.6", "kimi-k2.6"},
		{"glm-5.1", "glm-5.1"},
		{"mimo-v2.5", "mimo-v2.5"},
		{"minimax-m2.7", "minimax-m2.7"},
		{"qwen3.5-plus", "qwen3.5-plus"},
		// Prefixed variants → bare names
		{"opencode-go/deepseek-v4-flash", "deepseek-v4-flash"},
		{"deepseek/deepseek-v4-flash", "deepseek-v4-flash"},
		{"moonshotai/kimi-k2.6", "kimi-k2.6"},
		{"qwen3.6-plus", "qwen3.5-plus"},
		// Free-tier variants
		{"minimax-m2.5-free", "minimax/MiniMax-M2.5"},
		{"nemotron-3-super-free", "nvidia.nemotron-super-3-120b"},
		// Unknown model returns unchanged
		{"claude-sonnet-4-6", "claude-sonnet-4-6"},
		{"claude-opus-4-8", "claude-opus-4-8"},
		{"some-local-model", "some-local-model"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ResolveModelAlias(tt.input)
			if got != tt.want {
				t.Errorf("ResolveModelAlias(%q) = %q, want %q",
					tt.input, got, tt.want)
			}
		})
	}
}
