package az

import (
	"context"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	log "github.com/sirupsen/logrus"
)

// SubscriptionUser resolves the Username to stamp onto every subscription that
// this invocation enumerates.
//
// This replaces UserForTenant, which guessed by matching the cache entry's
// realm against the tenant. That guess was unsound: a live cache stores the
// literal realm "organizations" for every account, so with two identities
// cached it returned whichever one it happened to iterate last. Attribution
// instead follows the same precedence the token acquisition used, so the
// profile always credits the identity that actually made the API calls.
//
// An unresolvable identity yields an empty Username rather than an arbitrary
// one, because a blank user field is honest whereas a wrong one is a lie the
// user has no way to detect.
func SubscriptionUser(ctx context.Context, accounts []public.Account, hint string) string {
	active := ""
	if s, err := LoadState(ctx); err == nil {
		active = s.ActiveHomeAccountID
	}
	selected, err := ResolveAccount(accounts, hint, active, "")
	if err != nil {
		log.Debugf("subscription attribution left blank: %v", err)
		return ""
	}
	return selected.PreferredUsername
}
