package pricing

import "testing"

// TestModelAliases_NoEmptyTarget asserts that no alias entry maps to an
// empty string.
func TestModelAliases_NoEmptyTarget(t *testing.T) {
	for src, dst := range ModelAliases {
		if dst == "" {
			t.Errorf("ModelAliases[%q] = empty string; all targets must be non-empty", src)
		}
	}
}

// TestModelAliases_NoSelfMap asserts that no alias maps a key to itself.
func TestModelAliases_NoSelfMap(t *testing.T) {
	for src, dst := range ModelAliases {
		if src == dst {
			t.Errorf("ModelAliases[%q] = %q; alias must not map to itself", src, dst)
		}
	}
}

// TestModelAliases_Idempotency asserts that no alias target is itself an
// alias key, i.e. ResolveModelAlias is idempotent for all alias keys.
//
// If a target T is also an alias key, then ResolveModelAlias(ResolveModelAlias(k))
// != ResolveModelAlias(k), meaning one call is not enough to fully resolve the
// model name.
func TestModelAliases_Idempotency(t *testing.T) {
	for src := range ModelAliases {
		once := ResolveModelAlias(src)
		twice := ResolveModelAlias(once)
		if once != twice {
			t.Errorf(
				"alias chain detected: ResolveModelAlias(%q) = %q, "+
					"but ResolveModelAlias(%q) = %q; "+
					"alias target must not itself be an alias key",
				src, once, once, twice,
			)
		}
	}
}

// TestModelAliases_TargetResolvesToPricedEntry asserts that for every alias
// whose target is also a ModelPattern in FallbackPricing(), that target
// actually has a priced entry. This proves that the alias chain is
// well-formed: aliases pointing at a FallbackPricing key will actually
// resolve to real rates.
func TestModelAliases_TargetResolvesToPricedEntry(t *testing.T) {
	prices := FallbackPricing()
	fallbackPatterns := make(map[string]bool, len(prices))
	for _, p := range prices {
		fallbackPatterns[p.ModelPattern] = true
	}

	for src, dst := range ModelAliases {
		if !fallbackPatterns[dst] {
			// Target is a LiteLLM key, not a FallbackPricing key —
			// skip (these are fine; they resolve via LiteLLM DB rows).
			continue
		}
		// dst is a FallbackPricing ModelPattern. Verify it exists.
		found := false
		for _, p := range prices {
			if p.ModelPattern == dst {
				if p.InputPerMTok == 0 && p.OutputPerMTok == 0 {
					t.Errorf(
						"ModelAliases[%q] -> %q is in FallbackPricing "+
							"but has zero input and output rates",
						src, dst,
					)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf(
				"ModelAliases[%q] -> %q is a FallbackPricing pattern "+
					"but no entry was found",
				src, dst,
			)
		}
	}
}
