package az

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
)

func GetSignedInUser(ctx context.Context, tenant string) graphrbac.User {
	cclient := graphrbac.NewSignedInUserClient(tenant)
	cclient.Authorizer = GetAuthorizer(ctx, TokenOptions{
		policy.TokenRequestOptions{Scopes: []string{graphrbac.DefaultBaseURI + "/.default"}},
		AZ_CLIENT_ID,
		tenant,
	})
	u, err := cclient.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return u
}
