package az

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

var _ = Describe("Subscription attribution", func() {
	// Both accounts carry the realm "organizations", which is what a live cache
	// actually looks like. The old UserForTenant walked the cache and returned
	// whichever "organizations" entry it saw last, so a login as the second
	// identity could stamp the first identity's Username onto every
	// subscription. Attribution now comes from the Selected Account instead.
	first := acct("first@example.com", "oid-1", "tenant-a")
	second := acct("second@example.com", "oid-2", "tenant-b")
	pair := []public.Account{first, second}

	BeforeEach(func() {
		useTempCredDir()
	})

	It("attributes subscriptions to the active identity, not the other one", func() {
		Expect(StoreState(context.Background(), State{
			ActiveUsername:      "second@example.com",
			ActiveHomeAccountID: "oid-2.tenant-b",
		})).To(Succeed())

		Expect(SubscriptionUser(context.Background(), pair, "")).
			To(Equal("second@example.com"))
	})

	It("honours an explicit hint over the recorded active identity", func() {
		Expect(StoreState(context.Background(), State{
			ActiveUsername:      "second@example.com",
			ActiveHomeAccountID: "oid-2.tenant-b",
		})).To(Succeed())

		Expect(SubscriptionUser(context.Background(), pair, "first@example.com")).
			To(Equal("first@example.com"))
	})

	It("uses the sole cached identity when nothing is recorded", func() {
		Expect(SubscriptionUser(context.Background(), []public.Account{first}, "")).
			To(Equal("first@example.com"))
	})

	It("returns an empty user rather than guessing between identities", func() {
		Expect(SubscriptionUser(context.Background(), pair, "")).To(BeEmpty())
	})

	It("returns an empty user when the cache is empty", func() {
		Expect(SubscriptionUser(context.Background(), nil, "")).To(BeEmpty())
	})
})
