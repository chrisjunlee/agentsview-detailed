# Claude Opus 4.8 Usage Recognition Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make agentsview count and price Claude Opus 4.8 usage from Claude
Code transcripts.

**Architecture:** Claude transcript parsing already reads the structured
`message.model` field and token usage from `message.usage`. The missing behavior
is fallback pricing for `claude-opus-4-8` plus a fallback pricing version bump so
existing SQLite databases upsert the new pricing row on restart.

**Tech Stack:** Go, SQLite, existing `internal/pricing` fallback catalog,
existing `internal/db` usage aggregation, existing Claude JSONL parser tests.

---

## Resolved Facts

- Official Claude API and Vertex model ID: `claude-opus-4-8`.
- Official Bedrock model ID: `anthropic.claude-opus-4-8`.
- Official pricing: input `$5/MTok`, output `$25/MTok`, cache write
  `$6.25/MTok`, cache read `$0.50/MTok`.
- Official sources:
  - https://platform.claude.com/docs/en/about-claude/models/whats-new-claude-4-8
  - https://platform.claude.com/docs/en/about-claude/models/overview
  - https://platform.claude.com/docs/en/about-claude/models/model-ids-and-versions
- Local Claude transcript scan found `message.model == "claude-opus-4-8"` with
  `message.usage` present on the assistant messages.
- Do not scan content text for model detection.
- Do not add Bedrock, regional, or provider-prefixed aliases in this patch.

## Current TDD State

The tests for this change have already been added. Production implementation has
not been added.

Expected red tests before implementation:

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/pricing -run 'TestFallbackPricing|TestFallbackVersionBumpedForOpus48|TestParseLiteLLMPricing' -count=1
CGO_ENABLED=1 go test -tags fts5 ./internal/db -run TestGetDailyUsageUsesOpus48FallbackPricing -count=1
```

Expected green characterization tests before implementation:

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/parser -run TestParseClaudeSession_ExtractsMessageIDAndRequestID -count=1
CGO_ENABLED=1 go test -tags fts5 ./internal/pricing -run TestResolveModelAlias -count=1
```

## Files

- Modify: `internal/pricing/fallback.go`
- Already modified with tests: `internal/parser/claude_parser_test.go`
- Already modified with tests: `internal/pricing/fallback_test.go`
- Already modified with tests: `internal/pricing/litellm_test.go`
- Already modified with tests: `internal/pricing/aliases_test.go`
- Already modified with tests: `internal/db/usage_test.go`
- Do not modify frontend files, API handlers, DB migrations, or parser logic
  unless the parser characterization test fails.

## Task 1: Confirm The Red/Green Baseline

- [ ] **Step 1: Check repo state**

```bash
git status --short
```

Expected: test files and this plan may be modified. Do not overwrite unrelated
user changes.

- [ ] **Step 2: Run parser characterization**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/parser -run TestParseClaudeSession_ExtractsMessageIDAndRequestID -count=1
```

Expected: pass. If it fails, inspect
`internal/parser/claude.go::extractClaudeTokenFields` and ensure it reads
`message.model` into `msg.Model`.

- [ ] **Step 3: Run alias guard**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/pricing -run TestResolveModelAlias -count=1
```

Expected: pass. Do not add an alias for `claude-opus-4-8`; the test expects the
canonical ID to remain unchanged.

- [ ] **Step 4: Run pricing red tests**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/pricing -run 'TestFallbackPricing|TestFallbackVersionBumpedForOpus48|TestParseLiteLLMPricing' -count=1
```

Expected: fail because `claude-opus-4-8` is missing from fallback pricing and
`FallbackVersion` is still the old value.

- [ ] **Step 5: Run DB usage red test**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/db -run TestGetDailyUsageUsesOpus48FallbackPricing -count=1
```

Expected: fail because the seeded fallback pricing does not include
`claude-opus-4-8`, so the cost is zero instead of `$8.90`.

## Task 2: Implement Fallback Pricing

- [ ] **Step 1: Open `internal/pricing/fallback.go`**

Find:

```go
const FallbackVersion = "2026-05-03.1"
```

Change it to:

```go
const FallbackVersion = "2026-05-30.1"
```

- [ ] **Step 2: Add the Opus 4.8 pricing row**

In `FallbackPricing()`, add this entry near the current Claude model rows,
immediately after `claude-opus-4-6`:

```go
{
	ModelPattern:         "claude-opus-4-8",
	InputPerMTok:         5.0,
	OutputPerMTok:        25.0,
	CacheCreationPerMTok: 6.25,
	CacheReadPerMTok:     0.50,
},
```

- [ ] **Step 3: Format**

```bash
gofmt -w internal/pricing/fallback.go
```

- [ ] **Step 4: Re-run pricing tests**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/pricing -count=1
```

Expected: pass.

## Task 3: Prove DB Usage Now Prices Opus 4.8

- [ ] **Step 1: Run the focused DB test**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/db -run TestGetDailyUsageUsesOpus48FallbackPricing -count=1
```

Expected: pass.

- [ ] **Step 2: If cost is still zero**

Inspect `internal/db/usage.go::loadPricingMap`. Confirm it loads
`model_pricing.model_pattern` into the exact map key and that the test helper
seeded fallback pricing from `pricing.FallbackPricing()`.

- [ ] **Step 3: If the model does not appear in `ModelsUsed`**

Inspect the inserted test message. It must have:

```go
Model: "claude-opus-4-8"
TokenUsage: json.RawMessage(`{"input_tokens":1000000,...}`)
```

Do not normalize the model name.

## Task 4: Full Verification

- [ ] **Step 1: Format all Go files**

```bash
gofmt -w internal/parser/claude_parser_test.go \
         internal/pricing/fallback.go \
         internal/pricing/fallback_test.go \
         internal/pricing/litellm_test.go \
         internal/pricing/aliases_test.go \
         internal/db/usage_test.go
```

- [ ] **Step 2: Run targeted package tests**

```bash
CGO_ENABLED=1 go test -tags fts5 ./internal/parser ./internal/pricing ./internal/db -count=1
```

Expected: pass.

- [ ] **Step 3: Run required checks**

```bash
go vet ./...
make test-short
make build
```

Expected: all pass.

- [ ] **Step 4: Restart the embedded runtime**

```bash
pkill -f 'agentsview serve' 2>/dev/null; sleep 0.5
/Users/christopherlee/projects/pi-usage/agentsview/agentsview serve &
```

- [ ] **Step 5: Runtime smoke**

Call `/api/v1/usage/summary` for a date range that contains local Opus 4.8
Claude transcripts and confirm `claude-opus-4-8` has nonzero tokens and nonzero
cost.

If runtime cost remains zero after tests pass, check for a stale server process
or stale fallback seed. Confirm the SQLite `model_pricing` table has a
`claude-opus-4-8` row. Do not delete or recreate the database.

## Task 5: Commit

- [ ] **Step 1: Review changes**

```bash
git status --short
git diff
```

- [ ] **Step 2: Stage only intended files**

```bash
git add docs/superpowers/plans/2026-05-30-claude-opus-4-8-usage.md \
        internal/parser/claude_parser_test.go \
        internal/pricing/fallback.go \
        internal/pricing/fallback_test.go \
        internal/pricing/litellm_test.go \
        internal/pricing/aliases_test.go \
        internal/db/usage_test.go
```

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: recognize claude opus 4.8 usage"
```

## Self-Correction Rules

- Parser test fails: fix only `extractClaudeTokenFields` to read
  `message.model`; do not add a whitelist.
- Pricing tests fail: check exact model string `claude-opus-4-8`, exact rates,
  and exact `FallbackVersion`.
- DB usage test cost is zero: the fallback row is missing from the test DB or
  the model string does not match exactly.
- Runtime remains wrong after green tests: rebuild/restart first, then inspect
  pricing seed metadata. Do not modify parser logic unless transcript metadata
  proves parser extraction is failing.
