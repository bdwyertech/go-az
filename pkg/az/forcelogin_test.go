package az

import (
	"context"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RecordActiveAccount", func() {
	BeforeEach(func() { useTempCredDir() })

	It("records the authenticated identity", func() {
		a := acct("fresh@example.com", "oid-7", "tenant-c")
		Expect(RecordActiveAccount(context.Background(), a)).To(Succeed())

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s.ActiveHomeAccountID).To(Equal("oid-7.tenant-c"))
		Expect(s.ActiveUsername).To(Equal("fresh@example.com"))
	})

	It("leaves state untouched for an empty account", func() {
		Expect(RecordActiveAccount(context.Background(), public.Account{})).To(Succeed())

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(State{}))
	})
})

var _ = Describe("Forced interactive login", func() {
	BeforeEach(func() { useTempCredDir() })

	// A forced login must never be satisfied from the cache. With no browser
	// available the attempt has to fail rather than quietly return a cached
	// token, and the recorded active user must survive that failure.
	It("does not return a cached token and changes nothing on failure", func() {
		before := State{ActiveUsername: "prior@example.com", ActiveHomeAccountID: "oid-1.tenant-a"}
		Expect(StoreState(context.Background(), before)).To(Succeed())

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := GetToken(ctx, TokenOptions{ForceInteractive: true})
		Expect(err).To(HaveOccurred())

		after, lerr := LoadState(context.Background())
		Expect(lerr).NotTo(HaveOccurred())
		Expect(after).To(Equal(before))
	})
})
