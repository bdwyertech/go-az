package az

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"pgregory.net/rapid"
)

// Every property here runs at rapid's default of 100 checks, the minimum sample
// size the design calls for. Raise it locally with -rapid.checks.

// genSnapshot draws a non-empty account snapshot with unique home account ids,
// mirroring the shape MSAL writes into the token cache.
func genSnapshot(t *rapid.T) []public.Account {
	n := rapid.IntRange(1, 6).Draw(t, "accounts")
	out := make([]public.Account, 0, n)
	for i := 0; i < n; i++ {
		tenant := rapid.SampledFrom([]string{"tenant-a", "tenant-b", "tenant-c"}).
			Draw(t, fmt.Sprintf("tenant%d", i))
		user := rapid.SampledFrom([]string{"", "a@x.com", "b@y.com", "c@z.com"}).
			Draw(t, fmt.Sprintf("user%d", i))
		oid := fmt.Sprintf("oid-%d", i)
		out = append(out, public.Account{
			HomeAccountID:     oid + "." + tenant,
			LocalAccountID:    oid,
			Realm:             "organizations",
			PreferredUsername: user,
		})
	}
	return out
}

// contains reports whether the snapshot holds the given account.
func contains(accounts []public.Account, a public.Account) bool {
	for _, c := range accounts {
		if c.HomeAccountID == a.HomeAccountID {
			return true
		}
	}
	return false
}

// TestSelectionIsTotalAndClosed validates design Property 1.
func TestSelectionIsTotalAndClosed(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		accounts := genSnapshot(t)
		hint := rapid.SampledFrom([]string{"", "a@x.com", "oid-0", "nobody"}).Draw(t, "hint")
		tenant := rapid.SampledFrom([]string{"", "tenant-a", "tenant-z"}).Draw(t, "tenant")

		got, err := ResolveAccount(accounts, hint, "", tenant)
		if err != nil {
			if got.HomeAccountID != "" {
				t.Fatalf("error returned alongside account %q", got.HomeAccountID)
			}
			return
		}
		if !contains(accounts, got) {
			t.Fatalf("account %q is not in the snapshot", got.HomeAccountID)
		}
	})
}

// TestMatchingHintAlwaysWins validates design Property 2.
func TestMatchingHintAlwaysWins(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		accounts := genSnapshot(t)
		want := accounts[rapid.IntRange(0, len(accounts)-1).Draw(t, "index")]

		field := rapid.SampledFrom([]string{"username", "oid", "home"}).Draw(t, "field")
		hint := want.LocalAccountID
		switch field {
		case "username":
			if want.PreferredUsername == "" {
				return // an absent username is not a usable hint
			}
			hint = want.PreferredUsername
		case "home":
			hint = want.HomeAccountID
		}
		if rapid.Bool().Draw(t, "upper") {
			hint = strings.ToUpper(hint)
		}

		// A username may repeat across tenants, so assert the hint selects an
		// account bearing the hinted value rather than one specific index.
		got, err := ResolveAccount(accounts, hint, "oid-9.tenant-z", "tenant-z")
		if err != nil {
			t.Fatalf("hint %q was rejected: %v", hint, err)
		}
		if !matchesHint(got, hint) {
			t.Fatalf("hint %q selected unrelated account %q", hint, got.HomeAccountID)
		}
	})
}

// TestNonMatchingHintNeverSubstitutes validates design Property 3.
func TestNonMatchingHintNeverSubstitutes(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		accounts := genSnapshot(t)
		hint := rapid.SampledFrom([]string{"nobody@example.com", "oid-99", "oid-99.tenant-z"}).
			Draw(t, "hint")
		active := rapid.SampledFrom([]string{"", accounts[0].HomeAccountID}).Draw(t, "active")

		_, err := ResolveAccount(accounts, hint, active, "tenant-a")
		if !errors.Is(err, ErrNoMatchingAccount) {
			t.Fatalf("hint %q did not yield ErrNoMatchingAccount: %v", hint, err)
		}
	})
}

// TestSelectionIgnoresSnapshotOrder validates design Property 4.
func TestSelectionIgnoresSnapshotOrder(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		accounts := genSnapshot(t)
		hint := rapid.SampledFrom([]string{"", "a@x.com", "oid-0"}).Draw(t, "hint")
		active := rapid.SampledFrom([]string{"", accounts[0].HomeAccountID}).Draw(t, "active")
		tenant := rapid.SampledFrom([]string{"", "tenant-a", "tenant-b"}).Draw(t, "tenant")

		shuffled := append([]public.Account(nil), accounts...)
		rapid.Permutation(shuffled).Draw(t, "permutation")

		first, errA := ResolveAccount(accounts, hint, active, tenant)
		second, errB := ResolveAccount(shuffled, hint, active, tenant)
		if (errA == nil) != (errB == nil) {
			t.Fatalf("order changed the outcome: %v vs %v", errA, errB)
		}
		if errA == nil && first.HomeAccountID != second.HomeAccountID {
			t.Fatalf("order changed the account: %q vs %q",
				first.HomeAccountID, second.HomeAccountID)
		}
	})
}

// TestRealmNeverDecidesSelection validates design Property 5.
func TestRealmNeverDecidesSelection(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		accounts := genSnapshot(t)
		hint := rapid.SampledFrom([]string{"", "a@x.com", "oid-0"}).Draw(t, "hint")
		tenant := rapid.SampledFrom([]string{"", "tenant-a", "tenant-b"}).Draw(t, "tenant")

		rewritten := append([]public.Account(nil), accounts...)
		for i := range rewritten {
			rewritten[i].Realm = rapid.SampledFrom(
				[]string{"organizations", "common", "tenant-a", "tenant-z"},
			).Draw(t, fmt.Sprintf("realm%d", i))
		}

		first, errA := ResolveAccount(accounts, hint, "", tenant)
		second, errB := ResolveAccount(rewritten, hint, "", tenant)
		if (errA == nil) != (errB == nil) {
			t.Fatalf("realm changed the outcome: %v vs %v", errA, errB)
		}
		if errA == nil && first.HomeAccountID != second.HomeAccountID {
			t.Fatalf("realm changed the account: %q vs %q",
				first.HomeAccountID, second.HomeAccountID)
		}
	})
}

// TestActiveAccountIsHonouredWhenUnhinted validates design Property 6.
func TestActiveAccountIsHonouredWhenUnhinted(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		accounts := genSnapshot(t)
		i := rapid.IntRange(0, len(accounts)-1).Draw(t, "active")
		want := accounts[i]

		got, err := ResolveAccount(accounts, "", want.HomeAccountID, "")
		if err != nil {
			t.Fatalf("active account %q was not resolved: %v", want.HomeAccountID, err)
		}
		if got.HomeAccountID != want.HomeAccountID {
			t.Fatalf("active account ignored: got %q, want %q",
				got.HomeAccountID, want.HomeAccountID)
		}
	})
}
