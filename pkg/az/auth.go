package az

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// TokenCredential represents a credential capable of providing an OAuth token.
type TokenCredential struct {
	ClientID, TenantID string
	// PreferredUsername is the Account Hint for this credential. It selects
	// among cached accounts and never triggers a new sign-in on its own.
	PreferredUsername string
}

// GetToken requests an access token for the specified set of scopes.
func (c TokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// if options.TenantID == "" && c.TenantID != "" {
	// 	options.TenantID = c.TenantID
	// }
	token, err := GetToken(ctx, TokenOptions{
		TokenRequestOptions: options,
		ClientID:            c.ClientID,
		TenantID:            c.TenantID,
		PreferredUsername:   c.PreferredUsername,
	})
	if err != nil {
		return azcore.AccessToken{}, err
	}

	return azcore.AccessToken{
		Token:     token.AccessToken,
		ExpiresOn: token.ExpiresOn.UTC(),
	}, nil
}

type TokenOptions struct {
	policy.TokenRequestOptions
	ClientID, TenantID string
	// PreferredUsername is the Account Hint used to pick one cached account.
	PreferredUsername string
	// ForceInteractive skips every cache-only path so a fresh sign-in always
	// prompts. Existing cached accounts survive: MSAL adds to the cache rather
	// than replacing it, so a forced login never signs the other users out.
	ForceInteractive bool
}

// GetToken requests an access token for the specified set of scopes.
func GetToken(ctx context.Context, options TokenOptions) (token public.AuthResult, err error) {
	// Authority
	// https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-client-application-configuration#authority
	// Work & School Accounts - login.microsoftonline.com/organizations/
	// Specific Org Accounts - login.microsoftonline.com/<tenant-id>/
	if options.TenantID == "" {
		options.TenantID = "organizations"
	}

	if options.ClientID == "" {
		options.ClientID = AZ_CLIENT_ID
	}

	// GetToken takes options by value, but the Scopes slice header is shared with
	// the caller. Replace it with a private copy before anything downstream can
	// append through it.
	if len(options.Scopes) == 0 {
		options.Scopes = []string{
			azure.PublicCloud.ServiceManagementEndpoint + "/.default", // https://management.core.windows.net//.default
		}
	} else {
		options.Scopes = dedupeScopes(options.Scopes)
	}
	// Step 1: attempt a cache-only acquisition, unless the caller demanded a
	// fresh prompt. A forced login must not be satisfied by a cached token, so
	// the silent path is skipped rather than merely ignored on failure.
	if options.ForceInteractive {
		log.Debugln("Forced interactive login: skipping cached tokens")
	} else if token, err = acquireSilent(ctx, options); err == nil {
		return
	} else {
		log.Debugln("Silent token acquisition failed:", err.Error())
	}

	// Step 2: serialize interactive prompts across concurrent tooling.
	lockPath, lerr := interactiveLockPath()
	if lerr != nil {
		return token, lerr
	}
	log.Debugln("Acquiring interactive lock")

	// Step 3: retry silently while holding the lock, so a token written by
	// whichever caller won the race is observed instead of prompting again.
	err = withExclusiveLock(ctx, lockPath, func() error {
		if !options.ForceInteractive {
			if t, serr := acquireSilent(ctx, options); serr == nil {
				token = t
				return nil
			}
		}
		// Step 4: fall through to interactive only if that still fails.
		token, err = acquireInteractive(ctx, options)
		return err
	})
	return
}

// acquireInteractive performs the user-facing exchange, either via device code
// or a local browser redirect. It is only reached while the interactive lock is
// held, so at most one prompt is outstanding per credential directory.
func acquireInteractive(ctx context.Context, options TokenOptions) (token public.AuthResult, err error) {
	t := interactiveTransport()
	defer t.CloseIdleConnections()

	pubClient, err := newPubClient(options, t)
	if err != nil {
		return
	}

	if os.Getenv("GO_AZ_DEVICECODE") != "" {
		var code public.DeviceCode
		code, err = pubClient.AcquireTokenByDeviceCode(ctx, options.Scopes, public.WithTenantID(options.TenantID))
		if err != nil {
			return
		}
		log.Info(code.Result.Message)
		return code.AuthenticationResult(ctx)
	}

	// MSAL binds the redirect listener itself, so the port is reserved, released,
	// and handed over; a lost race is retried on a fresh port rather than failing
	// the whole login.
	err = withRedirectPort(ctx, func(port int) error {
		var berr error
		token, berr = pubClient.AcquireTokenInteractive(ctx, options.Scopes,
			public.WithRedirectURI(fmt.Sprintf("http://localhost:%v", port)),
			public.WithTenantID(options.TenantID))
		return berr
	})
	return
}

func GetAuthorizer(ctx context.Context, options TokenOptions) (*autorest.BearerAuthorizer, error) {
	token, err := GetToken(ctx, options)
	if err != nil {
		return nil, err
	}
	cliToken := cli.Token{
		AccessToken: token.AccessToken,
		ExpiresOn:   token.ExpiresOn.Format(time.RFC3339),
		TokenType:   "Bearer",
	}

	adalToken, err := cliToken.ToADALToken()
	if err != nil {
		return nil, err
	}

	oauthCfg, err := adal.NewOAuthConfig(microsoftAuthorityHost, token.IDToken.TenantID)
	if err != nil {
		return nil, err
	}

	t, err := adal.NewServicePrincipalTokenFromManualToken(*oauthCfg, token.IDToken.Audience, microsoftAuthorityHost, adalToken)
	if err != nil {
		return nil, err
	}
	return autorest.NewBearerAuthorizer(t), nil
}

type AccessToken struct {
	AccessToken  string `json:"accessToken"`
	ExpiresOn    string `json:"expiresOn"`
	Subscription string `json:"subscription,omitempty"`
	Tenant       string `json:"tenant"`
	TokenType    string `json:"tokenType"`
	// account is the identity MSAL actually authenticated. It is unexported so
	// the JSON stays byte-compatible with the Microsoft Azure CLI's output.
	account public.Account
}

// Account reports the identity this token was issued to.
func (t AccessToken) Account() public.Account {
	return t.account
}

type AccessTokenOptions struct {
	SubscriptionID string
	Resource       string
	ResourceType   string
	Scope          []string
	Tenant         string
	Client         string
	// PreferredUsername is the Account Hint for this invocation.
	PreferredUsername string
	// ForceInteractive demands a fresh sign-in even when a token is cached.
	ForceInteractive bool
}

func GetAccessToken(ctx context.Context, opts AccessTokenOptions) (token AccessToken, err error) {
	popts := policy.TokenRequestOptions{
		// withScope copies, so appending the resource scope cannot write back
		// into the caller's slice.
		Scopes: withScope(opts.Scope, opts.Resource),
		// TenantID: opts.Tenant,
	}

	t, err := GetToken(ctx, TokenOptions{
		TokenRequestOptions: popts,
		ClientID:            opts.Client,
		TenantID:            opts.Tenant,
		PreferredUsername:   opts.PreferredUsername,
		ForceInteractive:    opts.ForceInteractive,
	})
	if err != nil {
		return
	}
	token = AccessToken{
		AccessToken:  t.AccessToken,
		ExpiresOn:    t.ExpiresOn.Format("2006-01-02 15:04:05.000000"),
		Subscription: opts.SubscriptionID,
		Tenant:       t.IDToken.TenantID,
		TokenType:    "Bearer",
		account:      t.Account,
	}
	return
}

func GetCachedAccounts(ctx context.Context) (accounts []public.Account, err error) {
	pubClient, err := public.New(AZ_CLIENT_ID, public.WithCache(credCache))
	if err != nil {
		return
	}

	return pubClient.Accounts(ctx)
}
