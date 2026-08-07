package az

import (
	"sort"

	"github.com/Azure/go-autorest/autorest/azure/cli"
)

// MergeSubscriptions folds discovered into existing, keyed by subscription ID.
//
// A login by a second identity must not erase the subscriptions the first
// identity contributed, so the union is kept rather than the newest snapshot.
// Where an ID appears in both inputs the discovered entry wins, because it was
// just read from the API and carries the enumerating Username. The result is
// sorted by TenantID then ID so repeated logins produce a byte-identical
// profile, and exactly one entry is left default: the previous default when its
// subscription survives, otherwise the first of the sorted result.
func MergeSubscriptions(existing, discovered []cli.Subscription) []cli.Subscription {
	prevDefault := ""
	for _, s := range existing {
		if s.IsDefault {
			prevDefault = s.ID
			break
		}
	}

	byID := make(map[string]cli.Subscription, len(existing)+len(discovered))
	order := make([]string, 0, len(existing)+len(discovered))
	for _, s := range append(append([]cli.Subscription{}, existing...), discovered...) {
		if _, seen := byID[s.ID]; !seen {
			order = append(order, s.ID)
		}
		s.IsDefault = false
		byID[s.ID] = s
	}

	merged := make([]cli.Subscription, 0, len(order))
	for _, id := range order {
		merged = append(merged, byID[id])
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].TenantID != merged[j].TenantID {
			return merged[i].TenantID < merged[j].TenantID
		}
		return merged[i].ID < merged[j].ID
	})

	if len(merged) == 0 {
		return merged
	}
	def := 0
	for i := range merged {
		if merged[i].ID == prevDefault {
			def = i
			break
		}
	}
	merged[def].IsDefault = true
	return merged
}
