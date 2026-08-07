package az

import (
	"testing"

	"pgregory.net/rapid"
)

// scopeGen draws from a small alphabet so duplicates arise often.
func scopeGen(t *rapid.T, label string) []string {
	return rapid.SliceOfN(
		rapid.SampledFrom([]string{"a", "b", "c", "b/.default"}),
		0, 6,
	).Draw(t, label)
}

// TestScopesAreNeverMutated is Property 11: token options are never mutated.
//
// The interesting case is a caller slice with spare capacity, because that is
// when append reuses the caller's array. Rapid explores lengths, capacities and
// duplicate arrangements that a hand-written example would miss.
func TestScopesAreNeverMutated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		drawn := scopeGen(t, "scopes")
		slack := rapid.IntRange(0, 4).Draw(t, "slack")
		resource := rapid.SampledFrom([]string{"", "b", "https://vault.azure.net"}).Draw(t, "resource")

		caller := make([]string, len(drawn), len(drawn)+slack)
		copy(caller, drawn)

		got := withScope(caller, resource)

		if len(caller) != len(drawn) {
			t.Fatalf("caller length changed: got %d want %d", len(caller), len(drawn))
		}
		for i := range drawn {
			if caller[i] != drawn[i] {
				t.Fatalf("caller[%d] mutated: got %q want %q", i, caller[i], drawn[i])
			}
		}

		// Writing through the result must not reach the caller's array.
		for i := range got {
			got[i] = "sentinel"
		}
		for i := range drawn {
			if caller[i] != drawn[i] {
				t.Fatalf("result aliases caller at %d", i)
			}
		}
	})
}

// TestScopesAreDeduplicated pins the second half of the invariant: whatever the
// input arrangement, each scope appears exactly once in the result.
func TestScopesAreDeduplicated(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		drawn := scopeGen(t, "scopes")
		resource := rapid.SampledFrom([]string{"", "b", "c"}).Draw(t, "resource")

		got := withScope(drawn, resource)

		seen := map[string]bool{}
		for _, s := range got {
			if seen[s] {
				t.Fatalf("scope %q repeated in %v", s, got)
			}
			seen[s] = true
		}
		// Every input scope survives, and the resource scope is present.
		for _, s := range drawn {
			if !seen[s] {
				t.Fatalf("scope %q dropped from %v", s, got)
			}
		}
		if resource != "" && !seen[resource+"/.default"] {
			t.Fatalf("resource scope missing from %v", got)
		}
	})
}
