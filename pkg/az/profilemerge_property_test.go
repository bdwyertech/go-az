package az

import (
	"testing"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"pgregory.net/rapid"
)

// subGen draws a subscription list with a small ID and tenant alphabet so
// collisions between the two inputs are common rather than vanishingly rare.
func subGen(t *rapid.T, label string) []cli.Subscription {
	n := rapid.IntRange(0, 6).Draw(t, label+"-len")
	subs := make([]cli.Subscription, 0, n)
	for i := 0; i < n; i++ {
		subs = append(subs, cli.Subscription{
			ID:        rapid.SampledFrom([]string{"s1", "s2", "s3", "s4"}).Draw(t, label+"-id"),
			TenantID:  rapid.SampledFrom([]string{"t1", "t2"}).Draw(t, label+"-tenant"),
			Name:      label,
			IsDefault: rapid.Bool().Draw(t, label+"-default"),
		})
	}
	return subs
}

func idSet(subs []cli.Subscription) map[string]bool {
	m := map[string]bool{}
	for _, s := range subs {
		m[s.ID] = true
	}
	return m
}

// Property 7: merging preserves every subscription. No ID present in either
// input may be dropped, and no ID may be invented.
func TestMergePreservesEverySubscription(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		existing := subGen(t, "existing")
		discovered := subGen(t, "discovered")
		merged := MergeSubscriptions(existing, discovered)

		want := idSet(existing)
		for id := range idSet(discovered) {
			want[id] = true
		}
		got := idSet(merged)
		if len(got) != len(merged) {
			t.Fatalf("merge produced duplicate IDs: %v", merged)
		}
		for id := range want {
			if !got[id] {
				t.Fatalf("merge dropped subscription %q", id)
			}
		}
		for id := range got {
			if !want[id] {
				t.Fatalf("merge invented subscription %q", id)
			}
		}
	})
}

// Property 8: merging is idempotent. Re-merging the result with the same
// discovered set must not shuffle or change anything.
func TestMergeIsIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		existing := subGen(t, "existing")
		discovered := subGen(t, "discovered")
		once := MergeSubscriptions(existing, discovered)
		twice := MergeSubscriptions(once, discovered)
		if len(once) != len(twice) {
			t.Fatalf("length changed: %d then %d", len(once), len(twice))
		}
		for i := range once {
			if once[i] != twice[i] {
				t.Fatalf("entry %d changed: %+v then %+v", i, once[i], twice[i])
			}
		}
	})
}

// Property 9: merged order is deterministic. Shuffling either input cannot
// change the result, so the profile on disk stays byte-stable.
func TestMergeOrderIsDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		existing := subGen(t, "existing")
		discovered := subGen(t, "discovered")
		merged := MergeSubscriptions(existing, discovered)
		for i := 1; i < len(merged); i++ {
			a, b := merged[i-1], merged[i]
			if a.TenantID > b.TenantID || (a.TenantID == b.TenantID && a.ID >= b.ID) {
				t.Fatalf("unsorted at %d: %+v then %+v", i, a, b)
			}
		}
	})
}

// Property 10: exactly one default survives, and never zero, so the Azure CLI
// always has a subscription to fall back on.
func TestMergeKeepsExactlyOneDefault(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		existing := subGen(t, "existing")
		discovered := subGen(t, "discovered")
		merged := MergeSubscriptions(existing, discovered)
		defaults := 0
		for _, s := range merged {
			if s.IsDefault {
				defaults++
			}
		}
		if len(merged) == 0 {
			if defaults != 0 {
				t.Fatalf("empty merge claimed a default")
			}
			return
		}
		if defaults != 1 {
			t.Fatalf("want exactly 1 default, got %d in %+v", defaults, merged)
		}
	})
}
