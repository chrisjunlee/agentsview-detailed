# Pi Customization Test Suite — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax. Tasks 1–5 are independent and partitioned by package
> so they run as **parallel subagents writing disjoint new files**. Task 0 (main
> session) sets the baseline; Task 6 (main session) assembles and verifies.

**Goal:** Capture every behavioral customization in the 16 commits that
distinguish this fork (`main`) from the merge-base with `upstream/main` as an
automated test, so the work is regression-proof and the suite is a safety net for
the upcoming upstream rebase.

**Design spec:** `docs/superpowers/specs/2026-05-30-pi-customization-tests-design.md`

**Merge-base SHA:** `7490961` — every subagent inspects its slice of our work
with `git diff 7490961 main -- <files>`. That diff *is* the source of truth for
what behavior to characterize.

**Tech stack:** Go 1.26 (`CGO_ENABLED=1 -tags fts5`; postgres adds `pgtest`),
SQLite/FTS5, optional PostgreSQL (Docker confirmed up); Svelte 5 + vitest 4
(jsdom). Conventions: table-driven Go tests, `testDB(t)` helper, `t.TempDir()`, no
emojis, prefer stdlib, colocated `*.test.ts`.

---

## Resolved Facts (oracles — do not re-derive, do not guess)

### Pricing rates — `internal/pricing/fallback.go` vs `~/projects/api-cost-comparison/pricing.json`

Assert these EXACT values. USD per million tokens. `cache_write_5m` in the oracle
maps to `CacheCreationPerMTok`; `cache_read` maps to `CacheReadPerMTok`.

| ModelPattern | Input | Output | CacheCreation | CacheRead | Oracle status |
| --- | --- | --- | --- | --- | --- |
| `deepseek-v4-flash` | 0.14 | 0.28 | 0 | 0.0028 | matches oracle |
| `deepseek-v4-pro` | 1.74 | 3.48 | 0 | 0.0145 | matches oracle |
| `kimi-k2.6` | 0.95 | 4.00 | 0 | 0.16 | matches oracle |
| `glm-5.1` | 1.40 | 4.40 | 0 | 0.26 | matches oracle |
| `mimo-v2.5-pro` | 1.00 | 3.00 | 0 | 0.20 | matches oracle |
| `mimo-v2.5` | 0.40 | 2.00 | 0 | 0.08 | matches oracle |
| `minimax-m2.7` | 0.30 | 1.20 | 0.375 | 0.06 | matches oracle |
| `qwen3.5-plus` | 0.40 | 2.40 | 0 | 0.04 | oracle is banded; 0.04 = 10% of band-1 input 0.40 (cache_read_pct) |
| `mimo-v2-pro` | 1.00 | 3.00 | 0 | 0 | **NO oracle entry** — characterization only; add code comment |
| `google/gemma-4-31b-it` | 0.10 | 0.40 | 0 | 0 | **NO oracle entry** — characterization only; add code comment |
| `claude-sonnet-4-6` | 3.0 | 15.0 | 3.75 | 0.30 | matches oracle |
| `claude-haiku-4-5-20251001` | 1.0 | 5.0 | 1.25 | 0.10 | matches oracle `claude-haiku-4-5` |

`claude-opus-4-6` / `claude-opus-4-8` (5/25/6.25/0.50) and `FallbackVersion ==
"2026-05-30.1"` are already covered by `fallback_test.go` — do not duplicate.
Codex/`gpt-*` and historical `claude-*-2025*` rows have no oracle in pricing.json
(out of scope for exact assertions).

### Time-bound timestamp construction (identical in usage + analytics)

`UsageFilter.FromTimestamp()` / `AnalyticsFilter.utcRange()` "from":
`From + "T" + (FromTime+":00" if FromTime else "00:00:00") + "Z"`.
`ToTimestamp()` / utcRange "to":
`To + "T" + (ToTime+":59" if ToTime else "23:59:59") + "Z"`.
**Note the seconds:** an explicit `ToTime` gets `:59` seconds appended (so
`ToTime="14:30"` → `...T14:30:59Z`), while an absent ToTime uses `23:59:59`.

Examples to assert:
- From `2026-05-30`, no FromTime → `2026-05-30T00:00:00Z`
- From `2026-05-30`, FromTime `09:15` → `2026-05-30T09:15:00Z`
- To `2026-05-30`, no ToTime → `2026-05-30T23:59:59Z`
- To `2026-05-30`, ToTime `14:30` → `2026-05-30T14:30:59Z`

### `inLocalRange(ts, loc) (date string, ok bool)` (usage + analytics, identical)

1. If `ts` can't parse in `loc`: `date = ts[:10]` (or `""` if <10 chars), return
   `inDateRange(date, From, To)`.
2. Else `date = lt.Format("2006-01-02")`; if `!inDateRange(date,From,To)` → false.
3. If `FromTime != "" && date == From` and `lt.Format("15:04") < FromTime` → false
   (boundary: `== FromTime` passes).
4. If `ToTime != "" && date == To` and `lt.Format("15:04") > ToTime` → false
   (boundary: `== ToTime` passes).

### `timeutil.IsValidTime(s)` — `internal/timeutil/timeutil.go` (untested today)

True only for well-formed `HH:MM`, hour 0–23, minute 0–59, exactly 5 chars with
`:` at index 2. True: `00:00`, `09:15`, `23:59`. False: `24:00`, `12:60`, `1:30`,
`120:00`, `12:3`, `12:30:00`, `ab:cd`, ``, `12-30`.

### `projectColor(name)` — `frontend/src/lib/utils/projectColor.ts` (model-color branch untested)

Ordered regex list — order matters (assert the specific-before-generic cases):
- `claude-opus-4-6` → `#F34D3F` (NOT the generic claude orange — proves the
  opus-4-6 rule precedes the `^(claude|anthropic)` rule)
- `claude-opus-4-7` → `#D97606`
- `claude-haiku-3-5` → `#D97606`
- `gpt-5.4-mini` → `#0084C6` (proves it precedes the generic `^gpt` rule)
- `gpt-5.3-codex-spark` → `#0FA0C4`
- `gpt-5.4` → `#0FA0C4` (generic gpt/codex)
- `claude-sonnet-4-6` → `#D97606` (generic claude)
- `deepseek-v4-pro` → `#6A1B9A`
- `""` (empty) → `var(--text-muted)` (FALLBACK)
- a non-model name (e.g. `my-project`) → a `PROJECT_PALETTE` entry
  (`var(--accent-*)`), deterministic for the same input (call twice, assert equal)

### `STICKY_PARAMS` — `frontend/src/lib/stores/router.svelte.ts`

Now `new Set(["desktop", "story"])` (we added `"story"`). Navigations must
preserve `?story=1` and `?desktop=...`.

---

## Files

**Read-only sources** (never edit — characterize current behavior):
`internal/pricing/{aliases,fallback}.go`, `internal/db/{usage,analytics}.go`,
`internal/postgres/{usage,analytics}.go`, `internal/server/{usage,analytics,story,server}.go`,
`internal/timeutil/timeutil.go`, `frontend/src/lib/utils/projectColor.ts`,
`frontend/src/lib/stores/{usage,router,analytics}.svelte.ts`,
`frontend/src/lib/components/shared/{DateRangeSelector.svelte,dateRangeSelector.ts}`.

**New test files (one owner each — guarantees no concurrent edit collisions):**

| Subagent | New files |
| --- | --- |
| P Pricing | `internal/pricing/fallback_rates_test.go`, `internal/pricing/alias_integrity_test.go`, `internal/db/pricing_alias_resolution_test.go` |
| W Usage-window+timeutil | `internal/timeutil/isvalidtime_test.go`, `internal/db/usage_window_test.go`, `internal/server/usage_window_internal_test.go` |
| A Analytics | `internal/db/analytics_window_test.go`, `internal/server/analytics_window_internal_test.go`, `internal/postgres/analytics_window_pgtest_test.go`, `internal/postgres/usage_window_pgtest_test.go` |
| S Story-server | `internal/server/story_negotiation_test.go` |
| FE Frontend | `frontend/src/lib/utils/projectColor.modelcolors.test.ts`, `frontend/src/lib/stores/usage.window.test.ts`, `frontend/src/lib/stores/analytics.timefilter.test.ts`, `frontend/src/lib/stores/router.sticky.test.ts` |

Rule: do not edit any existing `_test.go` / `.test.ts`. If an existing test
already covers a behavior, note it and skip — do not duplicate. Each Go test file
must compile standalone using existing helpers (`testDB(t)`); a broken new file
breaks the whole package build for sibling subagents.

---

## TDD methodology (every test, no exceptions)

1. Write the characterization test asserting the EXACT expected value (oracle
   above where applicable).
2. Run it — it passes, because the production code already exists.
3. **Mutation check:** temporarily break the target (flip a rate, invert a
   boundary `<`→`<=`, drop the `"story"` sticky param) OR assert a deliberately
   wrong value, and confirm the test goes RED. Revert.
4. Only a RED-on-mutation test counts as real coverage. Behavior judged missing
   gets a true red→green cycle instead.

**Bug policy (design decision 4):** fix small/clear bugs and flag them; PAUSE and
flag anything that changes a user-visible cost number or is ambiguous; never
encode wrong behavior as "expected." The two no-oracle pricing rows are
characterization-only, explicitly commented.

---

## Task 0 — Baseline (main session, before dispatch)

- [ ] Confirm clean tree on `main`, Docker up, merge-base `7490961`.
- [ ] Record the green baseline:
  `CGO_ENABLED=1 go test -tags fts5 ./internal/pricing ./internal/db ./internal/timeutil ./internal/server -count=1`
  and `cd frontend && npm test`. Existing suite must be green before we add files.

## Task 1 — P · Pricing

- [ ] `git diff 7490961 main -- internal/pricing/aliases.go internal/pricing/fallback.go internal/db/usage.go` to see exactly what we added.
- [ ] `internal/pricing/fallback_rates_test.go`: table-driven, assert every row in
  the Resolved-Facts pricing table by exact `ModelPricing` value. Comment the two
  no-oracle rows. Do not re-test opus-4-6/4-8 or `FallbackVersion`.
- [ ] `internal/pricing/alias_integrity_test.go`: iterate `ModelAliases` and assert
  invariants — no empty target, no self-map (`src != dst`), and idempotency
  (`ResolveModelAlias(ResolveModelAlias(x)) == ResolveModelAlias(x)`, i.e. no alias
  target is itself an alias key). For targets that are bare names present in
  `FallbackPricing()`, assert they map to a priced entry.
- [ ] `internal/db/pricing_alias_resolution_test.go` (package `db`): end-to-end —
  read `aliases.go`, pick a `src→dst` where `dst` is in `FallbackPricing()` (e.g.
  an `opencode-*`/prefixed alias of `deepseek-v4-pro`); `testDB(t)`, seed fallback
  pricing, insert a message whose model is the alias `src`, run `GetDailyUsage`,
  assert the resulting cost equals the `dst` rates applied (non-zero, exact).
- [ ] Mutation-check each. Verify:
  `CGO_ENABLED=1 go test -tags fts5 ./internal/pricing ./internal/db -count=1`.

## Task 2 — W · Usage-window + timeutil

- [ ] `git diff 7490961 main -- internal/timeutil/timeutil.go internal/db/usage.go internal/server/usage.go`.
- [ ] `internal/timeutil/isvalidtime_test.go`: table-driven `TestIsValidTime` with
  the true/false cases above.
- [ ] `internal/db/usage_window_test.go` (package `db`): assert
  `UsageFilter.FromTimestamp()` / `ToTimestamp()` exact strings (incl. the `:59`
  ToTime case and the no-time defaults); assert `UsageFilter.inLocalRange` boundary
  logic (in-range, below-FromTime same-day rejects, above-ToTime same-day rejects,
  `==` boundaries pass, unparseable-ts date-only fallback). Use a fixed `loc`
  (e.g. `time.UTC` and one offset zone).
- [ ] `internal/server/usage_window_internal_test.go` (package `server`): drive
  `parseUsageFilter` (or the handler) — valid `from_time`/`to_time` populate the
  filter; invalid (`25:00`, `9:9`) → HTTP 400 with `use HH:MM`.
- [ ] Mutation-check each. Verify:
  `CGO_ENABLED=1 go test -tags fts5 ./internal/timeutil ./internal/db ./internal/server -count=1`.

## Task 3 — A · Analytics (SQLite + Postgres)

- [ ] `git diff 7490961 main -- internal/db/analytics.go internal/server/analytics.go internal/postgres/analytics.go internal/postgres/usage.go`.
- [ ] `internal/db/analytics_window_test.go` (package `db`): assert
  `AnalyticsFilter.utcRange()` exact from/to strings (with/without FromTime/ToTime,
  incl. `:59`) and `AnalyticsFilter.inLocalRange` boundary logic (mirror of the
  usage cases). These methods are unexported → same-package test.
- [ ] `internal/server/analytics_window_internal_test.go` (package `server`):
  `parseAnalyticsFilter` accepts valid `from_time`/`to_time`, 400s on invalid.
- [ ] `internal/postgres/analytics_window_pgtest_test.go` and
  `internal/postgres/usage_window_pgtest_test.go` (build tag `//go:build pgtest`):
  seed PG via existing pgtest helpers, exercise a time-bounded analytics + usage
  query, assert the time bound includes/excludes rows at the boundary. Mirror
  existing `*_pgtest_test.go` setup patterns.
- [ ] Mutation-check each. Verify:
  `CGO_ENABLED=1 go test -tags fts5 ./internal/db ./internal/server -count=1` and
  `make test-postgres` (or `TEST_PG_URL=... go test -tags "fts5,pgtest" ./internal/postgres/... -run Window -v`).

## Task 4 — S · Story-server

- [ ] `git diff 7490961 main -- internal/server/story.go internal/server/server.go`.
  Read `story_test.go` + `middleware_test.go` first and list what they already
  cover; only fill genuine gaps.
- [ ] `internal/server/story_negotiation_test.go`: focus on the pure/cheap units
  not requiring chromedp — `wantsBrowserPage` (Accept: text/html → true; JSON/empty
  → false), `parseStoryAppPath` (valid `/story`, `/story/...`, rejects), and
  content negotiation routing on `GET /api/v1/story` vs `GET /story`. Inject a fake
  `storyRenderer` via the `withStoryRenderer` Option so no real browser launches.
  Do NOT add a chromedp-dependent test. Confirm whether the screenshot
  PNG/`image/png` path and error cases are already covered before adding.
- [ ] Mutation-check. Verify:
  `CGO_ENABLED=1 go test -tags fts5 ./internal/server -count=1`.

## Task 5 — FE · Frontend units

- [ ] `git diff 7490961 main -- frontend/src/lib/utils/projectColor.ts frontend/src/lib/stores/usage.svelte.ts frontend/src/lib/stores/router.svelte.ts frontend/src/lib/stores/analytics.svelte.ts frontend/src/lib/components/shared/dateRangeSelector.ts`.
  Read existing `projectColor.test.ts`, `usage.test.ts`, `router.test.ts`,
  `analytics.test.ts`, `dateRangeSelector.test.ts`; skip already-covered behavior.
- [ ] `projectColor.modelcolors.test.ts`: the model-color cases above, asserting
  order sensitivity (opus-4-6 ≠ generic claude; gpt-5.4-mini ≠ generic gpt), empty
  → FALLBACK, non-model → deterministic palette entry.
- [ ] `usage.window.test.ts`: usage store noon-default window and explicit-one-day
  URL preservation (commits `d4dbc86`, `1e5c362`, `2b86c56`) — verify against the
  actual store API; if `usage.test.ts` already covers a case, skip it.
- [ ] `analytics.timefilter.test.ts`: `fromTime`/`toTime` default `"00:00"`;
  query params include `from_time`/`to_time` only when `!= "00:00"`;
  `setTimeRange` sets both; `setRollingWindow` resets both to `"00:00"`.
- [ ] `router.sticky.test.ts`: `STICKY_PARAMS` preserves `story` (and `desktop`)
  across a navigation.
- [ ] Mutation-check each. Verify: `cd frontend && npm test`.

## Task 6 — Assembly & verification (main session)

- [ ] Collect all new files; `gofmt -w` the Go ones; `go vet ./...`.
- [ ] Full Go suite: `CGO_ENABLED=1 go test -tags fts5 ./... -count=1`.
- [ ] Postgres: `make test-postgres` (Docker up) then `make postgres-down` when done.
- [ ] `make test-short`, `make build`, restart `agentsview serve` per CLAUDE.md.
- [ ] Frontend: `cd frontend && npm test`.
- [ ] Resolve any cross-file compile/conflict issues here in the main session.
- [ ] Stage explicitly (the new test files + both docs) and commit with a
  conventional message; never `git add -A`.
- [ ] Report flagged findings (any fallback/oracle divergence, any surfaced bug).

## Self-Correction Rules

- A new test file breaks package build for siblings → it must be self-contained
  and use existing helpers; fix the new file, don't touch existing ones.
- Pricing assertion fails → re-read `fallback.go`; the value in the table is the
  oracle. If code diverges from oracle on a user-visible cost, PAUSE and flag (do
  not edit the assertion to match wrong code).
- `inLocalRange`/`utcRange` exist in both usage and analytics — keep their tests in
  separate files (`usage_window_test.go` vs `analytics_window_test.go`); both in
  package `db` is fine.
- Story test wants a browser → use the injected fake `storyRenderer`; never launch
  chromedp in unit tests.
- pgtest fails to connect → ensure Docker PG is up (`make test-postgres`); these
  create/drop the `agentsview` schema, so only the test DB.
