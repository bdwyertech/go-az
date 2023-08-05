package az

import (
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
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
	"github.com/gofrs/flock"
	"github.com/skratchdot/open-golang/open"
)

//go:embed static/success.html
var okPage []byte

//go:embed static/fail.html
var failPage string

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

	token, err = pubClient.AcquireTokenSilent(ctx, options.Scopes, opts...)
	if err != nil {
		if strings.Contains(err.Error(), "AADSTS700082") || strings.Contains(err.Error(), "token_expired") || // Token Expired
			strings.Contains(err.Error(), "AADSTS50076") || // MFA Required
			strings.Contains(err.Error(), "AADSTS50079") || // MFA Enrollment Required
			// AADSTS50079: Due to a configuration change made by your administrator, or because you moved to a new location, you must enroll in multi-factor authentication
			strings.Contains(err.Error(), "AADSTS50173") || // Expired Grant
			strings.Contains(err.Error(), "AADSTS70043") { // Refresh Token Expired
			// AADSTS53003 - Access has been blocked by Conditional Access policies -- probably need to update OS or Browser
			//
			// http call(https://login.microsoftonline.com/organizations/oauth2/v2.0/token)(POST) error: reply status code was 400:
			// {"error":"invalid_grant","error_description":"AADSTS70043: The refresh token has expired or is invalid due to sign-in frequency checks by conditional access. The token was issued on 2022-01-15T22:57:51.2550000Z and the maximum allowed lifetime for this request is 32400.\r\nTrace ID: 05c52010-d810-4d78-91ca-c1318ad4ca00\r\nCorrelation ID: 6d2db73d-1006-47bb-a55b-1adb26ccc06e\r\nTimestamp: 2022-01-16 19:11:53Z","error_codes":[70043],"timestamp":"2022-01-16 19:11:53Z","trace_id":"05c52010-d810-4d78-91ca-c1318ad4ca00","correlation_id":"6d2db73d-1006-47bb-a55b-1adb26ccc06e","suberror":"token_expired"}
			//} else if err.Error() != "access token not found" && err.Error() != "no token found" && err.Error() != "not found" {
		} else {
			switch err.Error() {
			case "no token found",
				"access token not found",
				"not found",
				"no account was specified with public.WithAccount(), or the specified account is invalid":
			default:
				log.Debug(err.Error())
				return
			}
		}

		if !credCache.locked {
			// Tooling might call out concurrently -- ensure we only have one interactive prompt at any given time
			f := flock.New(filepath.Join(cacheDir(), ".go-az.lock"))
			log.Debug("Acquiring interactive lock")
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

		//
		// Device Code Flow
		//
		if os.Getenv("GO_AZ_DEVICECODE") != "" {
			var code public.DeviceCode
			code, err = pubClient.AcquireTokenByDeviceCode(ctx, options.Scopes)
			if err != nil {
				return
			}
			log.Info(code.Result.Message)
			return code.AuthenticationResult(ctx)
		}

		//
		// Authorization Code Flow
		//
		ictx, done := context.WithTimeout(ctx, 5*time.Minute)
		defer done()
		var callbackUrl string
		var code string
		var port int
		var listener net.Listener
		if os.Getenv("DEVWORKSPACE_METADATA") != "" {
			// Port has to be under 32000
			// https://github.com/che-incubator/che-code/blob/62888bf26329f8a53830a3e5c557e8efa2f3996c/code/extensions/che-port/src/ports-plugin.ts#LL165C28-L165C33
			port = 31999
			for {
				if listener, err = net.Listen("tcp", fmt.Sprintf(":%v", port)); err == nil {
					break
				}
				port = port - 1
			}
		} else {
			if listener, err = net.Listen("tcp", ":0"); err != nil {
				return
			}
			port = listener.Addr().(*net.TCPAddr).Port
		}
		defer listener.Close()

		if okPageb64 := os.Getenv("GO_AZ_OKPAGE_B64"); okPageb64 != "" {
			if okPage, err = base64.StdEncoding.DecodeString(okPageb64); err != nil {
				return
			}
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if headerErr := q.Get("error"); headerErr != "" {
				desc := q.Get("error_description")
				w.Write([]byte(fmt.Sprintf(failPage, headerErr, desc)))
				log.Errorln(headerErr, desc)
				return
			}
			if code = q.Get("code"); code != "" {
				w.Write(okPage)
				done()
				return
			}
			// Determine the callback URL from the first request
			if r.URL.Host == "" {
				r.URL.Host = fmt.Sprintf("localhost:%v", port)
				r.URL.Scheme = "http"
			}
			callbackUrl = r.URL.String()
			url, err := pubClient.AuthCodeURL(ctx, options.ClientID, callbackUrl, options.Scopes)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, err)
				return
			}
			http.Redirect(w, r, url, http.StatusFound)
		})
		go http.Serve(listener, mux)
		if os.Getenv("DEVWORKSPACE_METADATA") == "" {
			open.Start(fmt.Sprintf("http://localhost:%v", port))
		}
		<-ictx.Done()
		if err = ictx.Err(); !errors.Is(err, context.Canceled) {
			return
		}
		return pubClient.AcquireTokenByAuthCode(ctx, code, callbackUrl, options.Scopes)
	}

	return
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
