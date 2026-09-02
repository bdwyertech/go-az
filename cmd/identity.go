package cmd

import (
	"errors"
	"fmt"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
)

// resolveHint settles which cached identity this invocation runs as, before any
// token is requested.
//
// A token request carries the hint down into MSAL, where an unmatched
// --preferred-username surfaces as an interactive prompt or a token for the
// wrong identity rather than a clear error. Resolving first turns that into a
// clean abort: the caller learns the identity is wrong before the network is
// touched and before anything reaches stdout, which matters most for
// `kube-cred`, whose stdout is parsed by kubectl.
//
// The returned value is the canonical PreferredUsername from the cache rather
// than the raw hint, so the credential is built from the same spelling the cache
// uses.
//
// When the cache is completely empty, there is nothing to protect against, so
// the function returns the raw hint (possibly empty) and lets the token layer
// proceed to an interactive prompt.
func resolveHint(cmd *cobra.Command) (string, error) {
	hint := accountHint(cmd)
	username, err := resolveIdentity(cmd, hint)
	if err != nil {
		if errors.Is(err, az.ErrNoCachedAccounts) {
			return hint, nil
		}
		return "", fmt.Errorf("selecting an account: %w", err)
	}
	return username, nil
}
