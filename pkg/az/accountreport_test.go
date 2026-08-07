package az

import (
	"context"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Account reporting", func() {
	accounts := []public.Account{
		acct("first@example.com", "oid-1", "tenant-a"),
		acct("second@example.com", "oid-2", "tenant-b"),
	}

	BeforeEach(func() {
		useTempCredDir()
	})

	It("lists every cached account and names the active one", func() {
		Expect(StoreState(context.Background(), State{
			ActiveUsername:      "second@example.com",
			ActiveHomeAccountID: "oid-2.tenant-b",
		})).To(Succeed())

		r, err := BuildAccountReport(context.Background(), accounts)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.ActiveUsername).To(Equal("second@example.com"))
		Expect(r.ActiveHomeAccountID).To(Equal("oid-2.tenant-b"))
		Expect(r.Accounts).To(HaveLen(2))
		Expect(r.Accounts[0].Username).To(Equal("first@example.com"))
		Expect(r.Accounts[0].IsActive).To(BeFalse())
		Expect(r.Accounts[1].IsActive).To(BeTrue())
	})

	It("reports no active account when the state file is absent", func() {
		r, err := BuildAccountReport(context.Background(), accounts)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.ActiveUsername).To(BeEmpty())
		Expect(r.ActiveHomeAccountID).To(BeEmpty())
		for _, a := range r.Accounts {
			Expect(a.IsActive).To(BeFalse())
		}
	})

	It("marks a recorded account that is no longer cached as stale", func() {
		Expect(StoreState(context.Background(), State{
			ActiveUsername:      "gone@example.com",
			ActiveHomeAccountID: "oid-9.tenant-z",
		})).To(Succeed())

		r, err := BuildAccountReport(context.Background(), accounts)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.ActiveIsCached).To(BeFalse())
	})
})

var _ = Describe("SetActiveAccount", func() {
	accounts := []public.Account{
		acct("first@example.com", "oid-1", "tenant-a"),
		acct("second@example.com", "oid-2", "tenant-b"),
	}

	BeforeEach(func() {
		useTempCredDir()
	})

	It("records the matched account as active", func() {
		got, err := SetActiveAccount(context.Background(), accounts, "second@example.com")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.HomeAccountID).To(Equal("oid-2.tenant-b"))

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s.ActiveUsername).To(Equal("second@example.com"))
		Expect(s.ActiveHomeAccountID).To(Equal("oid-2.tenant-b"))
	})

	It("matches case-insensitively and by object id", func() {
		_, err := SetActiveAccount(context.Background(), accounts, "OID-1")
		Expect(err).NotTo(HaveOccurred())

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s.ActiveHomeAccountID).To(Equal("oid-1.tenant-a"))
	})

	It("leaves the recorded account unchanged when the hint matches nothing", func() {
		Expect(StoreState(context.Background(), State{
			ActiveUsername:      "first@example.com",
			ActiveHomeAccountID: "oid-1.tenant-a",
		})).To(Succeed())

		_, err := SetActiveAccount(context.Background(), accounts, "nobody@example.com")
		Expect(err).To(MatchError(ErrNoMatchingAccount))

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s.ActiveHomeAccountID).To(Equal("oid-1.tenant-a"))
	})

	It("rejects an empty hint without writing state", func() {
		_, err := SetActiveAccount(context.Background(), accounts, "")
		Expect(err).To(HaveOccurred())

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(State{}))
	})
})
