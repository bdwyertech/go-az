package az

import (
	"errors"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// acct builds a snapshot entry with the given username, object id and tenant.
func acct(username, oid, tenant string) public.Account {
	return public.Account{
		HomeAccountID:     oid + "." + tenant,
		LocalAccountID:    oid,
		Environment:       "login.microsoftonline.com",
		Realm:             "organizations",
		PreferredUsername: username,
	}
}

var _ = Describe("ResolveAccount", func() {
	a := acct("Brian.Dwyer@broadridge.com", "oid-1", "tenant-a")
	b := acct("DwyerAdminCld@Broadridge.onmicrosoft.com", "oid-2", "tenant-b")
	pair := []public.Account{a, b}

	It("returns ErrNoMatchingAccount for an empty snapshot", func() {
		_, err := ResolveAccount(nil, "", "", "")
		Expect(errors.Is(err, ErrNoMatchingAccount)).To(BeTrue())
	})

	It("matches a username hint without regard to case", func() {
		got, err := ResolveAccount(pair, "BRIAN.DWYER@BROADRIDGE.COM", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal(a.HomeAccountID))
	})

	It("matches an object id hint", func() {
		got, err := ResolveAccount(pair, "oid-2", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal(b.HomeAccountID))
	})

	It("matches a home account id hint", func() {
		got, err := ResolveAccount(pair, "oid-2.tenant-b", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal(b.HomeAccountID))
	})

	It("rejects a hint that matches nothing", func() {
		_, err := ResolveAccount(pair, "nobody@example.com", a.HomeAccountID, "")
		Expect(errors.Is(err, ErrNoMatchingAccount)).To(BeTrue())
	})

	It("prefers the active account when unhinted", func() {
		got, err := ResolveAccount(pair, "", b.HomeAccountID, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal(b.HomeAccountID))
	})

	It("falls back to a sole tenant match", func() {
		got, err := ResolveAccount(pair, "", "", "tenant-b")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal(b.HomeAccountID))
	})

	It("falls back to the only account", func() {
		got, err := ResolveAccount([]public.Account{a}, "", "", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal(a.HomeAccountID))
	})

	It("reports ambiguity when nothing narrows the snapshot", func() {
		_, err := ResolveAccount(pair, "", "", "")
		Expect(errors.Is(err, ErrAmbiguousAccount)).To(BeTrue())
		Expect(err.Error()).To(ContainSubstring(a.PreferredUsername))
	})

	Describe("deduplication", func() {
		// Same identity logged in through two authority endpoints produces two
		// cache entries that differ only in Realm.
		tenantSpecific := public.Account{
			HomeAccountID:     "oid-1.tenant-a",
			LocalAccountID:    "oid-1",
			Environment:       "login.microsoftonline.com",
			Realm:             "tenant-a",
			PreferredUsername: "Daniel.Ristic@broadridge.com",
		}
		orgRealm := public.Account{
			HomeAccountID:     "oid-1.tenant-a",
			LocalAccountID:    "oid-1",
			Environment:       "login.microsoftonline.com",
			Realm:             "organizations",
			PreferredUsername: "Daniel.Ristic@broadridge.com",
		}

		It("collapses two entries with the same home_account_id into one", func() {
			dupes := []public.Account{tenantSpecific, orgRealm}
			got, err := ResolveAccount(dupes, "", "", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.PreferredUsername).To(Equal("Daniel.Ristic@broadridge.com"))
		})

		It("prefers the tenant-specific realm over organizations", func() {
			// organizations first, tenant-specific second
			dupes := []public.Account{orgRealm, tenantSpecific}
			got, err := ResolveAccount(dupes, "", "", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Realm).To(Equal("tenant-a"))
		})

		It("does not collapse entries with different home_account_ids", func() {
			different := []public.Account{tenantSpecific, b}
			_, err := ResolveAccount(different, "", "", "")
			Expect(errors.Is(err, ErrAmbiguousAccount)).To(BeTrue())
		})
	})
})
