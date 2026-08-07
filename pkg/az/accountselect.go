package az

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	log "github.com/sirupsen/logrus"
)

// ErrNoMatchingAccount reports that the caller named an identity the local token
// cache does not hold.
var ErrNoMatchingAccount = errors.New("no cached account matches the requested user")

// ErrAmbiguousAccount reports that several cached identities are equally good
// candidates and the caller must say which one to use.
var ErrAmbiguousAccount = errors.New("several cached accounts match; specify one")

// homeTenant returns the tenant GUID encoded in the suffix of a home account id.
// MSAL writes home_account_id as "{object-id}.{tenant-id}". The realm field is
// unrelated: it is frequently the literal "organizations" and carries no tenant.
func homeTenant(a public.Account) string {
	if i := strings.LastIndex(a.HomeAccountID, "."); i >= 0 {
		return a.HomeAccountID[i+1:]
	}
	return ""
}

// ResolveAccount picks exactly one account from a single cache snapshot.
//
// The snapshot must be the one taken for this invocation, so that the account
// returned here is the same account a later token request will find.
func ResolveAccount(accounts []public.Account, hint, active, tenant string) (public.Account, error) {
	// Step 1: an explicit hint is authoritative. Matching it against a stale or
	// absent identity is an error rather than a silent fallback, because logging
	// in as somebody the caller did not ask for is worse than failing.
	if hint != "" {
		for _, a := range accounts {
			if matchesHint(a, hint) {
				log.Debugf("account %q selected by hint %q", a.PreferredUsername, hint)
				return a, nil
			}
		}
		return public.Account{}, fmt.Errorf("%w: %q", ErrNoMatchingAccount, hint)
	}

	// Step 2: the account the user last chose.
	for _, a := range accounts {
		if active != "" && strings.EqualFold(a.HomeAccountID, active) {
			log.Debugf("account %q selected as the active account", a.PreferredUsername)
			return a, nil
		}
	}

	// Step 3: a tenant narrows the field only when it narrows it to one.
	if tenant != "" {
		if sole, ok := soleTenantMatch(accounts, tenant); ok {
			log.Debugf("account %q selected as the only account in tenant %q",
				sole.PreferredUsername, tenant)
			return sole, nil
		}
	}

	// Step 4: no choice to make.
	if len(accounts) == 1 {
		log.Debugf("account %q selected as the only cached account",
			accounts[0].PreferredUsername)
		return accounts[0], nil
	}

	if len(accounts) == 0 {
		return public.Account{}, ErrNoMatchingAccount
	}
	return public.Account{}, fmt.Errorf("%w: %s", ErrAmbiguousAccount, strings.Join(usernames(accounts), ", "))
}

// matchesHint reports whether the hint names this account by username, object
// id, or home account id. Comparison is case-insensitive because Entra echoes
// usernames back in whatever case the user typed them.
func matchesHint(a public.Account, hint string) bool {
	return strings.EqualFold(a.PreferredUsername, hint) ||
		strings.EqualFold(a.LocalAccountID, hint) ||
		strings.EqualFold(a.HomeAccountID, hint)
}

// soleTenantMatch returns the single account homed in tenant, if there is
// exactly one.
func soleTenantMatch(accounts []public.Account, tenant string) (public.Account, bool) {
	var found public.Account
	n := 0
	for _, a := range accounts {
		if strings.EqualFold(homeTenant(a), tenant) {
			found = a
			n++
		}
	}
	return found, n == 1
}

// usernames lists the snapshot's identities for an error message, falling back
// to the home account id where a username is absent.
func usernames(accounts []public.Account) []string {
	out := make([]string, 0, len(accounts))
	for _, a := range accounts {
		if a.PreferredUsername != "" {
			out = append(out, a.PreferredUsername)
			continue
		}
		out = append(out, a.HomeAccountID)
	}
	return out
}
