package az

import (
	"strings"

	"github.com/Azure/go-autorest/autorest/azure/cli"
)

// FilterSubscriptionsByUser narrows a profile read to the subscriptions the
// named identity actually enumerated.
//
// azureProfile.json is a union across every identity that has ever logged in, so
// an unfiltered read hands one identity another's subscriptions and lets
// `account show` settle on a default that belongs to someone else. Filtering is
// read-side only: the file keeps both identities so the next invocation as the
// other one does not have to re-enumerate from nothing.
//
// An empty user means "no narrowing requested" and returns the input untouched,
// which preserves the behaviour of every caller that predates hints.
// Comparison is case-insensitive because Entra echoes UPN casing
// inconsistently and a case difference is never a different identity.
func FilterSubscriptionsByUser(subs []cli.Subscription, user string) []cli.Subscription {
	if user == "" {
		return subs
	}

	kept := make([]cli.Subscription, 0, len(subs))
	for _, s := range subs {
		if s.User != nil && strings.EqualFold(s.User.Name, user) {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		return kept
	}

	// The profile marks exactly one subscription default, and it may have just
	// been filtered out. Leaving the narrowed view with no default would make
	// `account show` report nothing even though this identity can see
	// subscriptions, so the first survivor is promoted.
	for _, s := range kept {
		if s.IsDefault {
			return kept
		}
	}
	kept[0].IsDefault = true
	return kept
}
