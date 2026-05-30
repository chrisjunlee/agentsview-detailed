/**
 * Characterization tests for the MODEL_COLORS ordered-rule table in
 * projectColor.ts.  These cover fork-specific additions; the existing
 * projectColor.test.ts covers the palette / fallback / determinism basics.
 */

import { describe, it, expect } from "vitest";
import { projectColor, PROJECT_PALETTE } from "./projectColor.js";

// ---------------------------------------------------------------------------
// 1. Each model regex maps to its documented hex color
// ---------------------------------------------------------------------------

describe("projectColor model-color rules — direct matches", () => {
  it("maps claude-opus-4-6 to #F34D3F (exact prefix)", () => {
    expect(projectColor("claude-opus-4-6")).toBe("#F34D3F");
  });

  it("maps claude-opus-4-6-20260601 to #F34D3F (prefix match)", () => {
    expect(projectColor("claude-opus-4-6-20260601")).toBe("#F34D3F");
  });

  it("maps CLAUDE-OPUS-4-6 to #F34D3F (case-insensitive)", () => {
    expect(projectColor("CLAUDE-OPUS-4-6")).toBe("#F34D3F");
  });

  it("maps claude-opus-4-7 to #D97606", () => {
    expect(projectColor("claude-opus-4-7")).toBe("#D97606");
  });

  it("maps claude-opus-4-7-20260601 to #D97606 (prefix match)", () => {
    expect(projectColor("claude-opus-4-7-20260601")).toBe("#D97606");
  });

  it("maps claude-haiku to #D97606", () => {
    expect(projectColor("claude-haiku")).toBe("#D97606");
  });

  it("maps claude-haiku-4-5 to #D97606 (prefix match)", () => {
    expect(projectColor("claude-haiku-4-5")).toBe("#D97606");
  });

  it("maps gpt-5.4-mini to #0084C6", () => {
    expect(projectColor("gpt-5.4-mini")).toBe("#0084C6");
  });

  it("maps gpt-5.4-mini-preview to #0084C6 (prefix match)", () => {
    expect(projectColor("gpt-5.4-mini-preview")).toBe("#0084C6");
  });

  it("maps GPT-5.4-MINI to #0084C6 (case-insensitive)", () => {
    expect(projectColor("GPT-5.4-MINI")).toBe("#0084C6");
  });

  it("maps gpt-5.3-codex-spark to #0FA0C4", () => {
    expect(projectColor("gpt-5.3-codex-spark")).toBe("#0FA0C4");
  });

  it("maps gpt-5.3-codex-spark-001 to #0FA0C4 (prefix match)", () => {
    expect(projectColor("gpt-5.3-codex-spark-001")).toBe("#0FA0C4");
  });

  it("maps generic gpt-4o to #0FA0C4 (falls through to gpt/o*/chatgpt rule)", () => {
    expect(projectColor("gpt-4o")).toBe("#0FA0C4");
  });

  it("maps o1-mini to #0FA0C4 (o[1-9] generic rule)", () => {
    expect(projectColor("o1-mini")).toBe("#0FA0C4");
  });

  it("maps o3 to #0FA0C4", () => {
    expect(projectColor("o3")).toBe("#0FA0C4");
  });

  it("maps chatgpt-4o to #0FA0C4", () => {
    expect(projectColor("chatgpt-4o")).toBe("#0FA0C4");
  });

  it("maps openai-davinci to #0FA0C4", () => {
    expect(projectColor("openai-davinci")).toBe("#0FA0C4");
  });

  it("maps codex-mini to #0FA0C4", () => {
    expect(projectColor("codex-mini")).toBe("#0FA0C4");
  });

  it("maps generic claude-sonnet-4-5 to #D97606 (claude generic rule)", () => {
    expect(projectColor("claude-sonnet-4-5")).toBe("#D97606");
  });

  it("maps anthropic-model to #D97606", () => {
    expect(projectColor("anthropic-model")).toBe("#D97606");
  });

  it("maps deepseek-v3 to #6A1B9A", () => {
    expect(projectColor("deepseek-v3")).toBe("#6A1B9A");
  });

  it("maps deepseek-coder to #6A1B9A", () => {
    expect(projectColor("deepseek-coder")).toBe("#6A1B9A");
  });

  it("maps DEEPSEEK-V3 to #6A1B9A (case-insensitive)", () => {
    expect(projectColor("DEEPSEEK-V3")).toBe("#6A1B9A");
  });
});

// ---------------------------------------------------------------------------
// 2. ORDER MATTERS — earlier specific rules must win over later generic ones
// ---------------------------------------------------------------------------

describe("projectColor model-color rules — order-matters proofs", () => {
  /**
   * claude-opus-4-6 matches BOTH /^claude-opus-4-6/i (rule 0, #F34D3F)
   * AND /^(claude|anthropic)/i (rule 6, #D97606).
   * The first rule must win.
   */
  it("claude-opus-4-6 resolves to #F34D3F, NOT the generic claude color #D97606", () => {
    const color = projectColor("claude-opus-4-6");
    expect(color).toBe("#F34D3F");
    expect(color).not.toBe("#D97606");
  });

  /**
   * claude-opus-4-7 matches BOTH /^claude-opus-4-7/i (rule 1, #D97606)
   * AND /^claude-haiku/i (rule 2) — except haiku won't match. But crucially
   * it also matches the generic /^(claude|anthropic)/i (rule 6, #D97606).
   * Same color here, but verify the specific rule fires before the generic.
   * We prove this by checking the color is correct AND that the haiku rule
   * does NOT intercept it (haiku rule color happens to be the same; the
   * ordering check is that claude-haiku does NOT return #F34D3F).
   */
  it("claude-haiku resolves to #D97606, NOT the claude-opus-4-6 color #F34D3F", () => {
    const color = projectColor("claude-haiku");
    expect(color).toBe("#D97606");
    expect(color).not.toBe("#F34D3F");
  });

  /**
   * gpt-5.4-mini matches BOTH /^gpt-5\.4-mini/i (rule 3, #0084C6)
   * AND /^(gpt|o[1-9]|chatgpt|openai|codex)/i (rule 5, #0FA0C4).
   * The specific rule must win → #0084C6, NOT #0FA0C4.
   */
  it("gpt-5.4-mini resolves to #0084C6, NOT the generic gpt color #0FA0C4", () => {
    const color = projectColor("gpt-5.4-mini");
    expect(color).toBe("#0084C6");
    expect(color).not.toBe("#0FA0C4");
  });

  /**
   * gpt-5.3-codex-spark matches BOTH /^gpt-5\.3-codex-spark/i (rule 4, #0FA0C4)
   * AND the generic gpt rule (rule 5, also #0FA0C4).
   * Color is identical so we verify via a name that uses a dot literal,
   * confirming the regex /^gpt-5\.4-mini/i requires a literal dot:
   * "gpt-5X4-mini" (non-dot) must NOT match rule 3 and instead fall to
   * generic rule 5.
   */
  it("gpt-5X4-mini (no literal dot) falls through to generic gpt rule #0FA0C4", () => {
    // "gpt-5X4-mini" won't match /^gpt-5\.4-mini/i because \. needs a dot
    const color = projectColor("gpt-5X4-mini");
    expect(color).toBe("#0FA0C4");
  });

  /**
   * "codex" starts with "codex" and matches the generic rule (rule 5, #0FA0C4).
   * It does NOT match the deepseek or claude rules.
   */
  it("codex resolves to #0FA0C4 via generic rule, not deepseek or claude", () => {
    const color = projectColor("codex");
    expect(color).toBe("#0FA0C4");
    expect(color).not.toBe("#6A1B9A");
    expect(color).not.toBe("#D97606");
  });

  /**
   * gpt-5.3-codex-spark matches rule 4 before rule 5 (both #0FA0C4).
   * The important ordering proof: gpt-5.3-codex-spark ALSO starts with "gpt"
   * (generic rule 5) — same color, but the order still matters because
   * if rules were reversed, gpt-5.3-codex-spark would still be correct.
   * We test it works regardless.
   */
  it("gpt-5.3-codex-spark resolves to #0FA0C4 (specific rule 4 or generic rule 5 same color)", () => {
    expect(projectColor("gpt-5.3-codex-spark")).toBe("#0FA0C4");
  });
});

// ---------------------------------------------------------------------------
// 3. Empty / blank input returns the muted fallback
// ---------------------------------------------------------------------------

describe("projectColor empty/blank fallback", () => {
  it("empty string returns var(--text-muted)", () => {
    expect(projectColor("")).toBe("var(--text-muted)");
  });

  it("whitespace-only string is NOT empty (treated as project name)", () => {
    // " " is truthy, so it goes through to palette (not the FALLBACK)
    const color = projectColor(" ");
    expect(color).not.toBe("var(--text-muted)");
    expect(PROJECT_PALETTE).toContain(color);
  });
});

// ---------------------------------------------------------------------------
// 4. Non-model project names: deterministic palette pick
// ---------------------------------------------------------------------------

describe("projectColor non-model project name determinism", () => {
  it("same name returns same palette color on repeated calls", () => {
    const c1 = projectColor("my-project");
    const c2 = projectColor("my-project");
    const c3 = projectColor("my-project");
    expect(c1).toBe(c2);
    expect(c2).toBe(c3);
  });

  it("project name produces a palette color, not a hex model color", () => {
    const color = projectColor("my-project");
    expect(PROJECT_PALETTE).toContain(color);
  });

  it("agentsview produces a stable palette color", () => {
    const color = projectColor("agentsview");
    expect(PROJECT_PALETTE).toContain(color);
    // Must be the same on a second call
    expect(projectColor("agentsview")).toBe(color);
  });

  it("project names that happen to start with non-model prefixes go to palette", () => {
    // "deepfoo" does NOT start with "deepseek"
    const color = projectColor("deepfoo");
    expect(PROJECT_PALETTE).toContain(color);
  });
});
