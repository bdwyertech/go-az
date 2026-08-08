package az

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hint propagation (Property 1)", func() {
	It("carries the hint to the tenants, subscriptions, and Graph credentials", func() {
		useTempCredDir()
		user, admin := twoIdentityAccounts()

		var rec credRecorder
		rec.install()

		e := NewEnumeratorForAccount(user.PreferredUsername)
		e.listSubs = func(context.Context, string) ([]*armsubscriptions.Subscription, error) { return nil, nil }
		_, _ = e.ListOrganizations(context.Background())

		for _, c := range append(append(rec.tenants, rec.subscriptions...), rec.graph...) {
			Expect(c.PreferredUsername).To(Equal(user.PreferredUsername))
			Expect(c.PreferredUsername).NotTo(Equal(admin.PreferredUsername))
		}
	})
})
