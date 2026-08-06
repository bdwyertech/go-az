package az

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

// interactiveLockPath is the advisory lock serializing interactive browser
// prompts. It is deliberately distinct from the token cache lock so waiting
// for a browser never blocks cache reads.
func interactiveLockPath() (string, error) {
	d, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "interactive_auth.lock"), nil
}

// silentTransport is used for non-interactive token acquisition, where
// connection reuse is desirable.
func silentTransport() *http.Transport {
	return cleanhttp.DefaultPooledTransport()
}

// interactiveTransport is used for interactive and device-code exchanges.
// DefaultTransport disables keepalives and connection pooling, which
// aggressive intercepting proxies handle far better. The transport is built
// fresh per attempt and never mutated after being handed to MSAL.
func interactiveTransport() *http.Transport {
	return cleanhttp.DefaultTransport()
}

// newPubClient builds an MSAL public client bound to the given transport.
func newPubClient(options TokenOptions, t *http.Transport) (public.Client, error) {
	return public.New(options.ClientID,
		public.WithCache(credCache),
		public.WithHTTPClient(&http.Client{Transport: t}),
		public.WithAuthority(fmt.Sprintf("https://login.microsoftonline.com/%s/", options.TenantID)),
	)
}

// acquireSilent attempts a cache-only token acquisition, preferring an account
// whose realm matches the requested tenant.
func acquireSilent(ctx context.Context, options TokenOptions) (public.AuthResult, error) {
	t := silentTransport()
	defer t.CloseIdleConnections()

	pubClient, err := newPubClient(options, t)
	if err != nil {
		return public.AuthResult{}, err
	}

	opts := []public.AcquireSilentOption{}
	if accounts, aerr := pubClient.Accounts(ctx); aerr == nil && len(accounts) > 0 {
		selected := &accounts[0]
		for i := range accounts {
			if accounts[i].Realm == options.TenantID {
				selected = &accounts[i]
				break
			}
		}
		opts = append(opts, public.WithSilentAccount(*selected))
	}
	opts = append(opts, public.WithTenantID(options.TenantID))

	return pubClient.AcquireTokenSilent(ctx, options.Scopes, opts...)
}
