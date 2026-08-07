package az

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// tenantDesc builds the shape the ARM tenants pager returns.
func tenantDesc(id string) *armsubscriptions.TenantIDDescription {
	return &armsubscriptions.TenantIDDescription{TenantID: &id}
}

// armSub builds an enabled subscription in the shape the ARM pager returns.
func armSub(id, name string) *armsubscriptions.Subscription {
	state := armsubscriptions.SubscriptionStateEnabled
	return &armsubscriptions.Subscription{
		SubscriptionID: &id,
		DisplayName:    &name,
		State:          &state,
	}
}

// fakeEnumerator wires an Enumerator to canned data so a spec can assert how
// many times each listing was reached rather than guessing from timings.
func fakeEnumerator(tenants []string, subs map[string][]*armsubscriptions.Subscription, accounts []public.Account) *Enumerator {
	e := NewEnumerator()
	e.listTenants = func(context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
		var out []*armsubscriptions.TenantIDDescription
		for _, t := range tenants {
			out = append(out, tenantDesc(t))
		}
		return out, nil
	}
	e.listSubs = func(_ context.Context, tenant string) ([]*armsubscriptions.Subscription, error) {
		return subs[tenant], nil
	}
	e.loadAccounts = func(context.Context) ([]public.Account, error) {
		return accounts, nil
	}
	return e
}

var _ = Describe("Bounded enumeration", func() {
	var ctx context.Context

	BeforeEach(func() {
		useTempCredDir()
		ctx = context.Background()
	})

	It("lists tenants once no matter how many callers ask", func() {
		e := fakeEnumerator([]string{"t1", "t2"}, nil, nil)

		_, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		_, err = e.ListTenantDetails(ctx)
		Expect(err).NotTo(HaveOccurred())
		_, err = e.Tenants(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(e.tenantCalls).To(Equal(1))
	})

	It("lists each tenant's subscriptions once per invocation", func() {
		subs := map[string][]*armsubscriptions.Subscription{
			"t1": {armSub("s1", "One")},
			"t2": {armSub("s2", "Two")},
		}
		e := fakeEnumerator([]string{"t1", "t2"}, subs, nil)

		_, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		_, err = e.ListTenantDetails(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(e.hasSubscriptions(ctx, "t1")).To(BeTrue())

		Expect(e.subCalls).To(Equal(map[string]int{"t1": 1, "t2": 1}))
	})

	It("parses the token cache once per invocation", func() {
		e := fakeEnumerator([]string{"t1"}, nil, []public.Account{acct("u@example.com", "oid", "t1")})

		_, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		_, err = e.Accounts(ctx)
		Expect(err).NotTo(HaveOccurred())
		_, err = e.Accounts(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(e.accountCalls).To(Equal(1))
	})

	It("does not retry a listing that already failed", func() {
		e := NewEnumerator()
		boom := errors.New("tenant unreachable")
		e.listTenants = func(context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
			return nil, boom
		}

		_, err := e.Tenants(ctx)
		Expect(err).To(MatchError(boom))
		_, err = e.Tenants(ctx)
		Expect(err).To(MatchError(boom))

		Expect(e.tenantCalls).To(Equal(1))
	})

	It("does not retry a tenant whose subscription listing failed", func() {
		e := NewEnumerator()
		boom := errors.New("forbidden")
		e.listSubs = func(context.Context, string) ([]*armsubscriptions.Subscription, error) {
			return nil, boom
		}

		_, err := e.Subscriptions(ctx, "t1")
		Expect(err).To(MatchError(boom))
		_, err = e.Subscriptions(ctx, "t1")
		Expect(err).To(MatchError(boom))

		Expect(e.subCalls).To(Equal(map[string]int{"t1": 1}))
	})

	It("keeps enumerating when one tenant is unreachable", func() {
		e := NewEnumerator()
		e.listTenants = func(context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
			return []*armsubscriptions.TenantIDDescription{tenantDesc("bad"), tenantDesc("good")}, nil
		}
		e.listSubs = func(_ context.Context, tenant string) ([]*armsubscriptions.Subscription, error) {
			if tenant == "bad" {
				return nil, errors.New("forbidden")
			}
			return []*armsubscriptions.Subscription{armSub("s1", "One")}, nil
		}
		e.loadAccounts = func(context.Context) ([]public.Account, error) { return nil, nil }

		got, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].ID).To(Equal("s1"))
	})

	It("emits one tenant-level entry per subscription-less tenant", func() {
		subs := map[string][]*armsubscriptions.Subscription{
			"t1": {armSub("s1", "One")},
		}
		e := fakeEnumerator([]string{"t1", "t2", "t3"}, subs, nil)

		got, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())

		// The accumulated-slice check used to drop t2 and t3 because t1 had
		// already contributed a subscription.
		Expect(got).To(HaveLen(3))
		Expect(got[1].ID).To(Equal("t2"))
		Expect(got[1].Name).To(Equal("N/A(tenant level account)"))
		Expect(got[2].ID).To(Equal("t3"))
	})

	It("tolerates absent optional fields", func() {
		e := NewEnumerator()
		e.listTenants = func(context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
			return []*armsubscriptions.TenantIDDescription{tenantDesc("t1"), {}}, nil
		}
		e.listSubs = func(context.Context, string) ([]*armsubscriptions.Subscription, error) {
			id := "s1"
			return []*armsubscriptions.Subscription{{SubscriptionID: &id}, {}}, nil
		}
		e.loadAccounts = func(context.Context) ([]public.Account, error) { return nil, nil }

		got, err := e.ListSubscriptions(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0].ID).To(Equal("s1"))
		Expect(got[0].Name).To(BeEmpty())
		Expect(got[0].State).To(BeEmpty())
	})
})
