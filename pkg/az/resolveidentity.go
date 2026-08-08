package az

import (
	"context"
	"fmt"
)

// ResolveEnumerationIdentity settles which cached identity an enumeration runs
// as, before the first API call and before the first byte of output.
//
// Resolving up front is what makes an unmatched or ambiguous hint a clean abort
// rather than a half-printed listing: the caller learns the identity is wrong
// while stdout is still empty. The returned value is the canonical
// PreferredUsername from the cache, not the raw hint, so every credential built
// from it and the attribution line the command prints agree on one spelling.
//
// An empty hint is not an error; it means "the Active Account", preserving the
// behaviour callers had before hints existed.
func ResolveEnumerationIdentity(ctx context.Context, e *Enumerator, hint string) (string, error) {
	accounts, err := e.Accounts(ctx)
	if err != nil {
		return "", fmt.Errorf("reading the token cache: %w", err)
	}

	// State is advisory: a missing or unreadable state file simply means there
	// is no Active Account to fall back to, which ResolveAccount reports as an
	// ambiguity if the cache holds more than one identity.
	var active string
	if s, serr := LoadState(ctx); serr == nil {
		active = s.ActiveHomeAccountID
	}

	account, err := ResolveAccount(accounts, hint, active, "")
	if err != nil {
		return "", err
	}
	return account.PreferredUsername, nil
}
