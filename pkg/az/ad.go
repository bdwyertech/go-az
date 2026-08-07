package az

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
)

// GetSignedInUser resolves the directory object for the identity this
// invocation authenticated as. Errors are returned rather than fatal so a
// caller can fall back or report cleanly.
func GetSignedInUser(ctx context.Context, tenant string) (graphrbac.User, error) {
	auth, err := GetAuthorizer(ctx, TokenOptions{
		TokenRequestOptions: policy.TokenRequestOptions{
			Scopes: []string{graphrbac.DefaultBaseURI + "/.default"},
		},
		ClientID: AZ_CLIENT_ID,
		TenantID: tenant,
	})
	if err != nil {
		return graphrbac.User{}, err
	}

	cclient := graphrbac.NewSignedInUserClient(tenant)
	cclient.Authorizer = auth
	return cclient.Get(ctx)
}
