package az

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	. "github.com/onsi/ginkgo/v2"
)

// credRecorder captures every TokenCredential handed to the ARM tenants
// client, the ARM subscriptions client, and the Graph client, so a spec can
// assert on PreferredUsername without a live network call.
type credRecorder struct {
	tenants       []TokenCredential
	subscriptions []TokenCredential
	graph         []TokenCredential
}

// install swaps the package-level client constructors for recording doubles
// and restores the originals during spec cleanup.
func (r *credRecorder) install() {
	prevTenants, prevSubs, prevGraph := newTenantsClient, newSubscriptionsClient, newGraphClient

	newTenantsClient = func(cred azcore.TokenCredential, options *arm.ClientOptions) (*armsubscriptions.TenantsClient, error) {
		r.tenants = append(r.tenants, cred.(TokenCredential))
		return prevTenants(cred, options)
	}
	newSubscriptionsClient = func(cred azcore.TokenCredential, options *arm.ClientOptions) (*armsubscriptions.Client, error) {
		r.subscriptions = append(r.subscriptions, cred.(TokenCredential))
		return prevSubs(cred, options)
	}
	newGraphClient = func(cred azcore.TokenCredential, scopes []string) (*msgraphsdk.GraphServiceClient, error) {
		r.graph = append(r.graph, cred.(TokenCredential))
		return prevGraph(cred, scopes)
	}

	DeferCleanup(func() {
		newTenantsClient, newSubscriptionsClient, newGraphClient = prevTenants, prevSubs, prevGraph
	})
}
