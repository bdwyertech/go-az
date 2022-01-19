package az

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	cookiejar "github.com/orirawlings/persistent-cookiejar"
)

// TokenCredential represents a credential capable of providing an OAuth token.
type TokenCredential struct {
	TenantID string
}

// GetToken requests an access token for the specified set of scopes.
func (c TokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (*azcore.AccessToken, error) {
	if options.TenantID == "" && c.TenantID != "" {
		options.TenantID = c.TenantID
	}
	token, err := GetToken(ctx, options)
	if err != nil {
		return nil, err
	}

	return &azcore.AccessToken{
		Token:     token.AccessToken,
		ExpiresOn: token.ExpiresOn.UTC(),
	}, nil
}

// GetToken requests an access token for the specified set of scopes.
func GetToken(ctx context.Context, options policy.TokenRequestOptions) (token public.AuthResult, err error) {
	jar, err := cookiejar.New(&cookiejar.Options{
		Filename:              filepath.Join(cacheDir(), "go_msal_cookie_cache.json"),
		PersistSessionCookies: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer jar.Save()

	// Authority
	// https://docs.microsoft.com/en-us/azure/active-directory/develop/msal-client-application-configuration#authority
	// Work & School Accounts - login.microsoftonline.com/organizations/
	// Specific Org Accounts - login.microsoftonline.com/<tenant-id>/
	if options.TenantID == "" {
		options.TenantID = "organizations"
	}

	pubClientOpts := []public.Option{
		public.WithCache(credCache),
		public.WithHTTPClient(&http.Client{Jar: jar}),
		public.WithAuthority(fmt.Sprintf("https://login.microsoftonline.com/%s/", options.TenantID)),
	}

	pubClient, err := public.New(AZ_CLIENT_ID, pubClientOpts...)
	if err != nil {
		log.Fatal(err)
	}

	//	options.Scopes
	//	options.TenantID

	//	scopes := []string{
	//		// azure.PublicCloud.ServiceManagementEndpoint + ".default", // https://management.core.windows.net/.default
	//		azure.PublicCloud.GraphEndpoint + ".default",
	//		// azure.PublicCloud.ServiceManagementEndpoint + ".default+offline_access+openid+profile",
	//		// azure.PublicCloud.ServiceManagementEndpoint + ".default+user_impersonation",
	//		// azure.PublicCloud.ServiceManagementEndpoint + "user_impersonation", // https://management.core.windows.net/user_impersonation
	//		// azure.PublicCloud.ServiceManagementEndpoint + "offline_access",     // https://management.core.windows.net/user_impersonation
	//		// azure.PublicCloud.ResourceManagerEndpoint + ".default",   // https://management.azure.com/.default
	//		// "offline_access", // Refresh Token
	//		// AZ_CLIENT_ID + "/.default", // CLI Defaults
	//	}
	if len(options.Scopes) == 0 {
		options.Scopes = []string{
			azure.PublicCloud.ServiceManagementEndpoint + "/.default", // https://management.core.windows.net//.default
		}
	}

	opts := []public.AcquireTokenSilentOption{}
	if cachedAccounts := pubClient.Accounts(); len(cachedAccounts) > 0 {
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

	token, err = pubClient.AcquireTokenSilent(ctx, options.Scopes, opts...)
	if err != nil {
		if strings.Contains(err.Error(), "token_expired") || // Token Expired
			strings.Contains(err.Error(), "AADSTS50076") { // MFA Required
			//
			// http call(https://login.microsoftonline.com/organizations/oauth2/v2.0/token)(POST) error: reply status code was 400:
			// {"error":"invalid_grant","error_description":"AADSTS70043: The refresh token has expired or is invalid due to sign-in frequency checks by conditional access. The token was issued on 2022-01-15T22:57:51.2550000Z and the maximum allowed lifetime for this request is 32400.\r\nTrace ID: 05c52010-d810-4d78-91ca-c1318ad4ca00\r\nCorrelation ID: 6d2db73d-1006-47bb-a55b-1adb26ccc06e\r\nTimestamp: 2022-01-16 19:11:53Z","error_codes":[70043],"timestamp":"2022-01-16 19:11:53Z","trace_id":"05c52010-d810-4d78-91ca-c1318ad4ca00","correlation_id":"6d2db73d-1006-47bb-a55b-1adb26ccc06e","suberror":"token_expired"}
		} else if err.Error() != "access token not found" && err.Error() != "not found" {
			log.Fatal(err)
		}
		//
		// AcquireTokenInteractive
		//
		var port int
		port, err = getFreePort()
		if err != nil {
			log.Fatal(err)
		}

		token, err = pubClient.AcquireTokenInteractive(ctx, options.Scopes, public.WithRedirectURI(fmt.Sprintf("http://localhost:%v", port)))
		if err != nil {
			log.Fatal(err)
		}
	}

	return
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

func GetAuthorizer(ctx context.Context, options policy.TokenRequestOptions) *autorest.BearerAuthorizer {
	token, err := GetToken(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	cliToken := cli.Token{
		AccessToken: token.AccessToken,
		ExpiresOn:   token.ExpiresOn.Format(time.RFC3339),
		TokenType:   "Bearer",
		//	Authority        string `json:"_authority"`
		//	ClientID         string `json:"_clientId"`
		//	ExpiresOn        string `json:"expiresOn"`
		//	IdentityProvider string `json:"identityProvider"`
		//	IsMRRT           bool   `json:"isMRRT"`
		//	RefreshToken     string `json:"refreshToken"`
		//	Resource         string `json:"resource"`
		//	TokenType        string `json:"tokenType"`
		//	UserID           string `json:"userId"`
	}

	adalToken, err := cliToken.ToADALToken()
	if err != nil {
		log.Fatal(err)
	}

	oauthCfg, err := adal.NewOAuthConfig(microsoftAuthorityHost, token.IDToken.TenantID)
	if err != nil {
		log.Fatal(err)
	}
	t, err := adal.NewServicePrincipalTokenFromManualToken(*oauthCfg, AZ_CLIENT_ID, microsoftAuthorityHost, adalToken)
	if err != nil {
		log.Fatal(err)
	}
	return autorest.NewBearerAuthorizer(t)
}

type AccessTokenOptions struct {
	SubscriptionID string
	Resource       string
	ResourceType   string
	Scope          []string
	Tenant         string
}

type AccessToken struct {
	AccessToken  string `json:"accessToken"`
	ExpiresOn    string `json:"expiresOn"`
	Subscription string `json:"subscription,omitempty"`
	Tenant       string `json:"tenant"`
	TokenType    string `json:"tokenType"`
}

func GetAccessToken(ctx context.Context, opts AccessTokenOptions) (token AccessToken, err error) {
	popts := policy.TokenRequestOptions{
		Scopes:   opts.Scope,
		TenantID: opts.Tenant,
	}
	if opts.Resource != "" {
		popts.Scopes = append(popts.Scopes, opts.Resource+"/.default")
	}

	t, err := GetToken(ctx, popts)
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

func GetCachedAccounts() []public.Account {
	pubClient, err := public.New(AZ_CLIENT_ID, public.WithCache(credCache))
	if err != nil {
		log.Fatal(err)
	}

	return pubClient.Accounts()
}
