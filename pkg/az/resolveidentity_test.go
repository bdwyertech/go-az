package az

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Unmatched and ambiguous hints abort before output (Property 5)", func() {
	// staticEnumerator returns a fake whose tenant and subscription listings
	// record whether they were reached, so a spec can prove resolution happens
	// strictly before any enumeration call.
	newProbe := func(accounts []public.Account, reached *bool) *Enumerator {
		e := NewEnumerator()
		e.loadAccounts = func(context.Context) ([]public.Account, error) { return accounts, nil }
		e.listTenants = func(context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
			*reached = true
			return nil, nil
		}
		e.listSubs = func(context.Context, string) ([]*armsubscriptions.Subscription, error) {
			*reached = true
			return nil, nil
		}
		return e
	}

	It("names the unmatched hint and never enumerates", func() {
		user, _ := twoIdentityAccounts()

		var reached bool
		e := newProbe([]public.Account{user}, &reached)

		_, err := ResolveEnumerationIdentity(context.Background(), e, "nobody@example.com")
		Expect(err).To(MatchError(ErrNoMatchingAccount))
		Expect(err.Error()).To(ContainSubstring("nobody@example.com"))
		Expect(reached).To(BeFalse())
	})

	It("returns the canonical username for a matching hint", func() {
		user, admin := twoIdentityAccounts()

		var reached bool
		e := newProbe([]public.Account{user, admin}, &reached)

		// A case-insensitive hint must still resolve to the cached spelling, so
		// downstream credentials and the attribution line agree.
		got, err := ResolveEnumerationIdentity(context.Background(), e, "brian.dwyer@BROADRIDGE.com")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(user.PreferredUsername))
		Expect(reached).To(BeFalse())
	})
})

var _ = Describe("Active Account is still the default (Property 4)", func() {
	It("resolves an empty hint to the recorded Active Account", func() {
		useTempCredDir()
		user, admin := twoIdentityAccounts()

		Expect(StoreState(context.Background(), State{
			ActiveUsername:      admin.PreferredUsername,
			ActiveHomeAccountID: admin.HomeAccountID,
		})).To(Succeed())

		e := NewEnumerator()
		e.loadAccounts = func(context.Context) ([]public.Account, error) {
			return []public.Account{user, admin}, nil
		}

		got, err := ResolveEnumerationIdentity(context.Background(), e, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(admin.PreferredUsername))
	})
})
