package az

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/Azure/go-autorest/autorest/azure/cli"
)

func ListSubscriptionsCLI(refresh bool) []cli.Subscription {
	if refresh {
		BuildProfile()
	}
	p, err := cli.ProfilePath()
	if err != nil {
		log.Fatal(err)
	}
	o, err := cli.LoadProfile(p)
	if err != nil {
		log.Fatal(err)
	}
	return o.Subscriptions
}

func ListSubscriptions() (subscriptions []cli.Subscription) {
	for _, t := range ListTenants() {
		for _, s := range ListSubscriptionsForTenant(*t.TenantID) {
			subscriptions = append(subscriptions, cli.Subscription{
				EnvironmentName: "AzureCloud",
				ID:              *s.SubscriptionID,
				IsDefault:       false,
				Name:            *s.DisplayName,
				State:           string(*s.State),
				TenantID:        *t.TenantID,
				User: &cli.User{
					Name: UserForTenant(*t.TenantID),
					Type: "user",
				},
			})
		}
	}
	return
}

// func ListSubscriptions() (subscriptions []*armsubscription.Subscription) {
// 	for _, t := range ListTenants() {
// 		subscriptions = append(subscriptions, ListSubscriptionsForTenant(*t.TenantID)...)
// 	}
// 	return
// }

func ListSubscriptionsForTenant(tenant string) (subscriptions []*armsubscription.Subscription) {
	client := armsubscription.NewSubscriptionsClient(TokenCredential{TenantID: tenant}, nil)
	pager := client.List(nil)
	for {
		subscriptions = append(subscriptions, pager.PageResponse().ListResult.Value...)
		if pager.NextPage(context.Background()) {
			continue
		}
		if err := pager.Err(); err != nil {
			log.Fatalf("failed to advance page: %v", err)
		}
		break
	}

	return
}

func ListTenants() (tenants []*armsubscription.TenantIDDescription) {
	client := armsubscription.NewTenantsClient(new(TokenCredential), nil)
	pager := client.List(nil)
	for {
		tenants = append(tenants, pager.PageResponse().Value...)
		if pager.NextPage(context.Background()) {
			continue
		}
		if err := pager.Err(); err != nil {
			log.Fatalf("failed to advance page: %v", err)
		}
		break
	}

	return
}
