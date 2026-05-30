# Pi Customization Test Suite — Design Spec

**Date:** 2026-05-30
**Status:** Approved (autonomous `/loop` directive; defaults ratified, no overrides)
**Topic:** Comprehensive unit-test coverage for the 16 commits that distinguish
this fork (`main`) from `upstream/main` (`wesm/agentsview`).

## Goal

Capture every behavioral customization in our 16 commits as an automated test so
that (a) our work is regression-proof and (b) the suite is the safety net for the
upcoming upstream rebase (currently 56 behind; collides on opus-4-8 fallback
pricing, upstream #565 vs our `af51834`).

## Scope

**In scope** — behavioral/logic changes in:

- **Pricing** — Pi fallback rates, alias resolution, opus-4-8 (`internal/pricing`,
  `internal/db`)
- **Usage window** — one-day window, noon default, explicit-URL preservation
  (`internal/timeutil`, `internal/db`, `internal/server`; FE usage/router stores)
- **Date/time filter** — native time picker, filter preservation across tab
  switches (FE `DateRangeSelector`, usage store)
- **Story screenshot mode** — endpoint, content negotiation, model colors
  (`internal/server`; FE `projectColor`)
- **Analytics backend** — SQLite + Postgres analytics queries, handler
  (`internal/db`, `internal/postgres`, `internal/server`; FE analytics store)

**Out of scope** — docs/handoff/plan commits, pure markup with no logic,
generated files (`go.sum`), exhaustive Playwright UI (≤2 story screenshot specs
only).

## Decisions (ratified)

1. **Layers** — Go unit (pure + SQLite via `testDB`) + frontend vitest,
   comprehensive. Postgres-tag tests included and runnable (Docker confirmed up).
   E2e limited to the story visual.
2. **TDD on existing code** — characterization tests; every test verified to fail
   when its target behavior is broken (mutation check). Existing passing tests are
   kept; new test files are added to avoid edit collisions.
3. **Pricing oracle** — assert exact Pi rates sourced from
   `~/projects/api-cost-comparison/pricing.json`; add a referential-integrity test
   that every alias target resolves to a real priced entry.
4. **Bugs surfaced** — fix small/clear bugs and flag them; pause on anything that
   changes user-visible cost numbers or is ambiguous; never encode wrong behavior
   as expected.
5. **Coverage bar** — every source/logic behavior; exclude docs/markup/generated.

## Coverage matrix

| Theme | Our source changes | Existing tests | Gaps to fill |
| --- | --- | --- | --- |
| Pricing | `pricing/aliases.go`, `pricing/fallback.go` (+77), `db/usage.go` (`loadPricingMap`) | `aliases_test.go`, `fallback_test.go` (opus-4-6/4-8 rates, version), `db/usage_test.go` (opus-4-8) | exact rates for every non-opus Pi model vs `pricing.json`; alias referential integrity; alias applied end-to-end for prefixed names |
| Usage window | `timeutil.go` (+10), `db/usage.go` (+94), `server/usage.go` (+15), `stores/usage.svelte.ts`, `stores/router.svelte.ts` | `usage.test.ts` (+93), `router.test.ts` (+12) | timeutil noon/day-window helper (none); db one-day window query; server param parsing; store noon-default + explicit-URL preservation |
| Date/time filter | `DateRangeSelector.svelte` (+51), `dateRangeSelector.ts` | `dateRangeSelector.test.ts` (+8) | time-picker parse/format; filter preservation across tab switch |
| Story mode | `server/story.go` (488 new), `server/server.go` (+74), `projectColor.ts` (+19), `AppHeader.svelte` (+22) | `story_test.go` (+389), `middleware_test.go` (+54) | `projectColor` model-color logic (none); verify story endpoint/param/error coverage is complete; content-negotiation cases |
| Analytics | `db/analytics.go` (+98), `postgres/analytics.go` (+81), `postgres/usage.go` (+12), `server/analytics.go` (+15), `stores/analytics.svelte.ts` (+14) | none in our diff | db analytics behaviors (SQLite); server handler; postgres analytics + usage (pgtest); analytics store |

## TDD-on-existing-code methodology

For each target behavior: write a characterization test asserting the exact
expected value (oracle-derived where applicable) → run it (passes, since the code
exists) → confirm it is a real test by mutating the source or asserting a
deliberately wrong value to observe RED → revert. Behavior we judge to be missing
gets a true red-green cycle.

## Subagent decomposition

Five parallel subagents partitioned by package, each writing **new dedicated**
`_test.go` / `.test.ts` files so concurrent work never edits the same file:

- **P** Pricing · **W** Usage-window + timeutil · **A** Analytics (incl. pgtest) ·
  **S** Story-server · **FE** Frontend units.

Main session keeps: assembly, full-suite verification, conflict resolution, commit.

## Verification

- Go: `CGO_ENABLED=1 go test -tags fts5 ./... -count=1`; postgres adds
  `-tags "fts5,pgtest"`; plus `make test-short`, `go vet ./...`.
- FE: `cd frontend && npm test` (vitest).

## Risks

- `db/usage_test.go` is shared by the pricing and window themes → mitigated by
  separate new test files per subagent.
- Postgres tests create/drop the `agentsview` schema → run only against the test DB.
- Pinning exact rates may reveal that `fallback.go` diverges from `pricing.json` →
  that is a finding to flag (decision 4), not silently encode.
