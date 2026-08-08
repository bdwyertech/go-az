package az

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Active Account independence (Property 3)", func() {
	It("enumerates as the hinted user regardless of the recorded Active Account", func() {
		useTempCredDir()
		user, admin := twoIdentityAccounts()

		Expect(StoreState(context.Background(), State{
			ActiveUsername:      admin.PreferredUsername,
			ActiveHomeAccountID: admin.HomeAccountID,
		})).To(Succeed())

		var rec credRecorder
		rec.install()

		e := NewEnumeratorForAccount(user.PreferredUsername)
		_, _ = e.Tenants(context.Background())

		for _, c := range rec.tenants {
			Expect(c.PreferredUsername).To(Equal(user.PreferredUsername))
		}
	})
})
