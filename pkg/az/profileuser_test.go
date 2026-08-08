package az

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Property 7: Profile user attribution is consistent. Every subscription an
// enumeration writes must be credited to the identity that actually acquired
// the tokens. A profile that attributes a subscription to the wrong identity is
// worse than one that leaves the field blank, because the user has no way to
// tell it is wrong.
var _ = Describe("Profile user attribution", func() {
	var ctx context.Context
	var user, admin public.Account

	// Two tenants with a subscription each, plus a tenant with none, so the
	// spec covers both the normal and the tenant-level-account branch.
	subs := map[string][]*armsubscriptions.Subscription{
		"tenant-a": {armSub("sub-a", "Sub A")},
		"tenant-b": {armSub("sub-b", "Sub B")},
	}
	tenants := []string{"tenant-a", "tenant-b", "tenant-empty"}

	BeforeEach(func() {
		ctx = context.Background()
		useTempCredDir()
		user, admin = twoIdentityAccounts()
		// Record the admin as the Active Account so a hint that names the
		// regular user has to override it rather than agree with it.
		Expect(StoreState(ctx, State{
			ActiveUsername:      admin.PreferredUsername,
			ActiveHomeAccountID: admin.HomeAccountID,
		})).To(Succeed())
	})

	It("credits every subscription to the hinted identity", func() {
		e := fakeEnumerator(tenants, subs, []public.Account{user, admin})
		e.hint = user.PreferredUsername

		got, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).NotTo(BeEmpty())

		for _, s := range got {
			Expect(s.User).NotTo(BeNil())
			Expect(s.User.Name).To(Equal(user.PreferredUsername))
		}
	})

	It("credits the Active Account when no hint is given", func() {
		e := fakeEnumerator(tenants, subs, []public.Account{user, admin})

		got, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).NotTo(BeEmpty())

		for _, s := range got {
			Expect(s.User).NotTo(BeNil())
			Expect(s.User.Name).To(Equal(admin.PreferredUsername))
		}
	})
})
