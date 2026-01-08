package az

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
)

func GetSignedInUser(ctx context.Context, tenant string) graphrbac.User {
	cclient := graphrbac.NewSignedInUserClient(tenant)
	cclient.Authorizer = GetAuthorizer(ctx, &TokenOptions{
		TokenRequestOptions: policy.TokenRequestOptions{
			Scopes:   []string{azure.PublicCloud.GraphEndpoint + "/.default"},
			TenantID: tenant,
		},
		ClientID:          AZ_CLIENT_ID,
		ForceInteractive:  false,
		PreferredUsername: "",
	})
	u, err := cclient.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return u
}
