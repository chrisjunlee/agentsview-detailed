export const PROJECT_PALETTE: readonly string[] = [
  "var(--accent-blue)",
  "var(--accent-purple)",
  "var(--accent-amber)",
  "var(--accent-teal)",
  "var(--accent-rose)",
  "var(--accent-green)",
  "var(--accent-indigo)",
  "var(--accent-orange)",
  "var(--accent-sky)",
  "var(--accent-pink)",
  "var(--accent-coral)",
  "var(--accent-lime)",
] as const;

const FALLBACK = "var(--text-muted)";

const MODEL_COLORS: Array<{
  pattern: RegExp;
  color: string;
}> = [
  { pattern: /^claude-opus-4-6/i, color: "#F34D3F" },
  { pattern: /^claude-opus-4-7/i, color: "#D97606" },
  { pattern: /^claude-haiku/i, color: "#D97606" },
  { pattern: /^gpt-5\.4-mini/i, color: "#0084C6" },
  { pattern: /^gpt-5\.3-codex-spark/i, color: "#0FA0C4" },
  { pattern: /^(gpt|o[1-9]|chatgpt|openai|codex)/i, color: "#0FA0C4" },
  { pattern: /^(claude|anthropic)/i, color: "#D97606" },
  { pattern: /^deepseek/i, color: "#6A1B9A" },
];

function djb2(s: string): number {
  let h = 5381;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) + h + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

export function projectColor(name: string): string {
  if (!name) return FALLBACK;
  for (const entry of MODEL_COLORS) {
    if (entry.pattern.test(name)) {
      return entry.color;
    }
  }
  return PROJECT_PALETTE[djb2(name) % PROJECT_PALETTE.length]!;
}
