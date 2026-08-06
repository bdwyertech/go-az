package az

import (
	"context"
	"fmt"
	"net"
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
}

// GetToken requests an access token for the specified set of scopes.
func (c TokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (azcore.AccessToken, error) {
	// if options.TenantID == "" && c.TenantID != "" {
	// 	options.TenantID = c.TenantID
	// }
	token, err := GetToken(ctx, TokenOptions{options, c.ClientID, c.TenantID})
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

	if len(options.Scopes) == 0 {
		options.Scopes = []string{
			azure.PublicCloud.ServiceManagementEndpoint + "/.default", // https://management.core.windows.net//.default
		}
	}
	// Step 1: attempt a cache-only acquisition.
	if token, err = acquireSilent(ctx, options); err == nil {
		return
	}
	log.Debugln("Silent token acquisition failed:", err.Error())

	// Step 2: serialize interactive prompts across concurrent tooling.
	lockPath, lerr := interactiveLockPath()
	if lerr != nil {
		return token, lerr
	}
	log.Debugln("Acquiring interactive lock")

	// Step 3: retry silently while holding the lock, so a token written by
	// whichever caller won the race is observed instead of prompting again.
	err = withExclusiveLock(ctx, lockPath, func() error {
		if t, serr := acquireSilent(ctx, options); serr == nil {
			token = t
			return nil
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

	var port int
	port, err = getFreePort()
	if err != nil {
		return
	}

	return pubClient.AcquireTokenInteractive(ctx, options.Scopes, public.WithRedirectURI(fmt.Sprintf("http://localhost:%v", port)), public.WithTenantID(options.TenantID))
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func GetAuthorizer(ctx context.Context, options TokenOptions) *autorest.BearerAuthorizer {
	token, err := GetToken(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	cliToken := cli.Token{
		AccessToken: token.AccessToken,
		ExpiresOn:   token.ExpiresOn.Format(time.RFC3339),
		TokenType:   "Bearer",
	}

	adalToken, err := cliToken.ToADALToken()
	if err != nil {
		log.Fatal(err)
	}

	oauthCfg, err := adal.NewOAuthConfig(microsoftAuthorityHost, token.IDToken.TenantID)
	if err != nil {
		log.Fatal(err)
	}

	t, err := adal.NewServicePrincipalTokenFromManualToken(*oauthCfg, token.IDToken.Audience, microsoftAuthorityHost, adalToken)
	if err != nil {
		log.Fatal(err)
	}
	return autorest.NewBearerAuthorizer(t)
}

type AccessToken struct {
	AccessToken  string `json:"accessToken"`
	ExpiresOn    string `json:"expiresOn"`
	Subscription string `json:"subscription,omitempty"`
	Tenant       string `json:"tenant"`
	TokenType    string `json:"tokenType"`
}

type AccessTokenOptions struct {
	SubscriptionID string
	Resource       string
	ResourceType   string
	Scope          []string
	Tenant         string
	Client         string
}

func GetAccessToken(ctx context.Context, opts AccessTokenOptions) (token AccessToken, err error) {
	popts := policy.TokenRequestOptions{
		Scopes: opts.Scope,
		// TenantID: opts.Tenant,
	}
	if opts.Resource != "" {
		popts.Scopes = append(popts.Scopes, opts.Resource+"/.default")
	}

	t, err := GetToken(ctx, TokenOptions{popts, opts.Client, opts.Tenant})
	if err != nil {
		return
	}
	token = AccessToken{
		AccessToken:  t.AccessToken,
		ExpiresOn:    t.ExpiresOn.Format("2006-01-02 15:04:05.000000"),
		Subscription: opts.SubscriptionID,
		Tenant:       t.IDToken.TenantID,
		TokenType:    "Bearer",
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
