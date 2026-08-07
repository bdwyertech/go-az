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
})
