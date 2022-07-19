package az

import (
	"context"
	"errors"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/Azure/go-autorest/autorest/azure/cli"
)

func ListSubscriptionsCLI(refresh bool) []cli.Subscription {
	p, err := cli.ProfilePath()
	if err != nil {
		log.Fatal(err)
	}
	if _, err = os.Stat(p); errors.Is(err, os.ErrNotExist) || refresh {
		BuildProfile()
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

func ListSubscriptionsForTenant(tenant string) (subscriptions []*armsubscription.Subscription) {
	client, err := armsubscription.NewSubscriptionsClient(TokenCredential{TenantID: tenant}, nil)
	if err != nil {
		log.Fatal(err)
	}
	pager := client.NewListPager(nil)
	for pager.More() {
		nextResult, err := pager.NextPage(context.Background())
		if err != nil {
			log.Fatalln("failed to advance page:", err)
		}
		subscriptions = append(subscriptions, nextResult.Value...)
	}
	// TODO: Ensure we only return "enabled" subscriptions
	return
}

func ListTenants() (tenants []*armsubscription.TenantIDDescription) {
	client, err := armsubscription.NewTenantsClient(TokenCredential{}, nil)
	if err != nil {
		log.Fatal(err)
	}
	pager := client.NewListPager(nil)
	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			log.Fatalln("failed to advance page:", err)
		}
		tenants = append(tenants, resp.Value...)
	}

	return
}
