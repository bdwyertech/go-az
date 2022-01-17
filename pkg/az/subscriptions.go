package az

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/Azure/go-autorest/autorest/azure/cli"
)

func ListSubscriptionsCLI() []cli.Subscription {
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

func ListSubscriptions() (subscriptions []*armsubscription.Subscription) {
	client := armsubscription.NewSubscriptionsClient(TokenCredential{}, nil)
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

func ListSubscriptionTenants() (tenants []*armsubscription.TenantIDDescription) {
	client := armsubscription.NewTenantsClient(TokenCredential{}, nil)
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
	// o, err := json.MarshalIndent(tenants, "", "  ")
	// if err != nil {
	// 	log.Fatal()
	// }
	// fmt.Println(string(o))
	// log.Fatal()

	return
}
