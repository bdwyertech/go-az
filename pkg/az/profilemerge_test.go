package az

import (
	"github.com/Azure/go-autorest/autorest/azure/cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func sub(id, tenant, user string) cli.Subscription {
	return cli.Subscription{
		ID:       id,
		TenantID: tenant,
		Name:     id,
		User:     &cli.User{Name: user, Type: "user"},
	}
}

var _ = Describe("MergeSubscriptions", func() {
	It("retains subscriptions the other identity contributed", func() {
		existing := []cli.Subscription{sub("s1", "t1", "alice@example.com")}
		discovered := []cli.Subscription{sub("s2", "t1", "bob@example.com")}

		merged := MergeSubscriptions(existing, discovered)

		Expect(merged).To(HaveLen(2))
		Expect(merged[0].ID).To(Equal("s1"))
		Expect(merged[0].User.Name).To(Equal("alice@example.com"))
		Expect(merged[1].User.Name).To(Equal("bob@example.com"))
	})

	It("lets the freshly discovered entry replace the stale one", func() {
		existing := []cli.Subscription{sub("s1", "t1", "alice@example.com")}
		discovered := []cli.Subscription{sub("s1", "t1", "bob@example.com")}

		merged := MergeSubscriptions(existing, discovered)

		Expect(merged).To(HaveLen(1))
		Expect(merged[0].User.Name).To(Equal("bob@example.com"))
	})

	It("orders by tenant then subscription regardless of input order", func() {
		existing := []cli.Subscription{sub("s9", "t2", "alice@example.com")}
		discovered := []cli.Subscription{
			sub("s3", "t1", "bob@example.com"),
			sub("s1", "t1", "bob@example.com"),
		}

		merged := MergeSubscriptions(existing, discovered)

		Expect([]string{merged[0].ID, merged[1].ID, merged[2].ID}).
			To(Equal([]string{"s1", "s3", "s9"}))
	})

	It("keeps the chosen default when its subscription survives the merge", func() {
		chosen := sub("s9", "t2", "alice@example.com")
		chosen.IsDefault = true
		discovered := []cli.Subscription{sub("s1", "t1", "bob@example.com")}

		merged := MergeSubscriptions([]cli.Subscription{chosen}, discovered)

		Expect(merged[0].ID).To(Equal("s1"))
		Expect(merged[0].IsDefault).To(BeFalse())
		Expect(merged[1].ID).To(Equal("s9"))
		Expect(merged[1].IsDefault).To(BeTrue())
	})

	It("promotes the first entry when no previous default survives", func() {
		merged := MergeSubscriptions(nil, []cli.Subscription{
			sub("s2", "t1", "bob@example.com"),
			sub("s1", "t1", "bob@example.com"),
		})

		Expect(merged[0].ID).To(Equal("s1"))
		Expect(merged[0].IsDefault).To(BeTrue())
		Expect(merged[1].IsDefault).To(BeFalse())
	})

	It("returns an empty list without claiming a default", func() {
		Expect(MergeSubscriptions(nil, nil)).To(BeEmpty())
	})
})
