package pricing

import "testing"

func TestResolveModelAlias(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"deepseek-v4-flash", "deepseek/deepseek-v3.2"},
		{"kimi-k2.5", "moonshot/kimi-k2.5"},
		{"glm-5.1", "zai/glm-5"},
		{"qwen/qwen3-coder:free", "openrouter/qwen/qwen3-coder"},
		{"meta-llama/llama-3.3-70b-instruct:free", "deepinfra/meta-llama/Llama-3.3-70B-Instruct"},
		// Unknown model returns unchanged
		{"claude-sonnet-4-6", "claude-sonnet-4-6"},
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
