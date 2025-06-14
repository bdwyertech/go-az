package az

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/gofrs/flock"
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

	t := http.DefaultTransport.(*http.Transport).Clone()
	pubClientOpts := []public.Option{
		public.WithCache(credCache),
		public.WithHTTPClient(&http.Client{Transport: t}),
		public.WithAuthority(fmt.Sprintf("https://login.microsoftonline.com/%s/", options.TenantID)),
	}

	if options.ClientID == "" {
		options.ClientID = AZ_CLIENT_ID
	}

	pubClient, err := public.New(options.ClientID, pubClientOpts...)
	if err != nil {
		return
	}

	if len(options.Scopes) == 0 {
		options.Scopes = []string{
			azure.PublicCloud.ServiceManagementEndpoint + "/.default", // https://management.core.windows.net//.default
		}
	}
	opts := []public.AcquireSilentOption{}
	if cachedAccounts, err := pubClient.Accounts(ctx); err == nil && len(cachedAccounts) > 0 {
		var selected *public.Account
		for _, a := range cachedAccounts {
			if a.Realm == options.TenantID {
				selected = &a
				break
			}
		}

		if selected == nil {
			selected = &cachedAccounts[0]
		}

		opts = append(opts, public.WithSilentAccount(*selected))
	}
	opts = append(opts, public.WithTenantID(options.TenantID))
	// We need to try to AcquireTokenSilent again because another process holding the lock may have returned successfully
	if token, err = pubClient.AcquireTokenSilent(ctx, options.Scopes, opts...); err == nil {
		return
	} else if !credCache.locked {
		log.Debugln("Silent token aquisition failed, proceeding to Interactive:", err.Error())
		// Tooling might call out concurrently -- ensure we only have one interactive prompt at any given time
		f := flock.New(filepath.Join(cacheDir(), ".go-az.lock"))
		log.Debugln("Acquiring interactive lock")
		if _, err = f.TryLockContext(ctx, time.Duration(rand.Intn(5000)+1000)*time.Millisecond); err != nil {
			return
		}
		defer f.Unlock()
		credCache.locked = true
		return GetToken(ctx, options)
	}

	//
	// AcquireTokenInteractive
	//

	// Keepalives do not play nice with aggressive proxies here
	t.DisableKeepAlives = true
	defer func() { t.DisableKeepAlives = false }()

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
