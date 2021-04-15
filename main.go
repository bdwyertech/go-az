package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/services/subscription/mgmt/2020-09-01/subscription"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	// "github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// Azure CLI Default Client ID & Tenant
// https://github.com/Azure/azure-cli/blob/7bcf85939f00fbfc1961b9a82f0584e17e33e577/src/azure-cli-core/azure/cli/core/_profile.py#L69
const AZ_CLIENT_ID = "04b07795-8ddb-461a-bbee-02f9e1bf7b46"

//
// // Authority: https://login.microsoftonline.com/common
// // Specific Tenant: https://login.microsoftonline.com/my-tenant-id
// const AZ_COMMON_TENANT = "common"
//
// var (
// 	publicClientApp *msal.PublicClientApplication
// 	scopes          = []string{"user.read"}
// )

func main() {
	pubClient, err := public.New(AZ_CLIENT_ID)
	if err != nil {
		log.Fatal(err)
	}

	port, err := getFreePort()
	if err != nil {
		log.Fatal(err)
	}
	redirectUrl := fmt.Sprintf("http://localhost:%v", port)
	token, err := pubClient.AcquireTokenInteractive(context.Background(), []string{
		azure.PublicCloud.ServiceManagementEndpoint + ".default", // https://management.core.windows.net/.default
		// azure.PublicCloud.ResourceManagerEndpoint + ".default",   // https://management.azure.com/.default
		// "offline_access", // Refresh Token
		// AZ_CLIENT_ID + "/.default", // CLI Defaults
	}, public.WithRedirectURI(redirectUrl))
	if err != nil {
		log.Fatal(err)
	}

	//Refresh Tokens???
	// https://github.com/Azure/azure-cli/blob/dev/src/azure-cli-core/azure/cli/core/_profile.py#L636

	jsonBytes, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(jsonBytes))

	// log.Printf("%#v\n", token.Account)
	// log.Printf("%#v\n", token.IDToken)
	// log.Printf("%#v\n", token.AccessToken)

	cliToken := cli.Token{
		AccessToken: token.AccessToken,
		ExpiresOn:   token.ExpiresOn.Format("2006-01-02T15:04:05Z07:00"),
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

	//https://github.com/Azure/go-autorest/blob/7ac73d3561eaa034f458f97362b2743e8b3c048e/autorest/adal/config.go
	const activeDirectoryEndpoint = "https://login.microsoftonline.com/"
	oauthCfg, err := adal.NewOAuthConfig(activeDirectoryEndpoint, token.IDToken.TenantID)
	if err != nil {
		log.Fatal(err)
	}
	t, err := adal.NewServicePrincipalTokenFromManualToken(*oauthCfg, AZ_CLIENT_ID, activeDirectoryEndpoint, adalToken)
	if err != nil {
		log.Fatal(err)
	}

	// log.Fatalf("%#v", t)

	tBytes, err := t.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(string(tBytes))

	authorizer := autorest.NewBearerAuthorizer(&adalToken)

	subClient := subscription.NewSubscriptionsClient()
	subClient.Authorizer = authorizer
	subsIterator, err := subClient.List(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	var subscriptions []subscription.Model
	for {
		subscriptions = append(subscriptions, subsIterator.Values()...)
		if subsIterator.NotDone() {
			subsIterator.Next()
			continue
		}
		break
	}

	for _, sub := range subscriptions {
		log.Println(*sub.DisplayName)
		log.Println(*sub.ID)
		log.Println(*sub.SubscriptionID)
		log.Println(sub.State)
		log.Println(*sub.SubscriptionPolicies.LocationPlacementID)
		log.Println(*sub.SubscriptionPolicies.QuotaID)
		log.Println(sub.SubscriptionPolicies.SpendingLimit)
		log.Println(*sub.AuthorizationSource)
	}
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
