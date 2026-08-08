package az

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Property 8: Profile reads are scoped to the resolved identity.
//
// azureProfile.json is a union of every identity that has ever enumerated, so an
// unfiltered read shows one identity another identity's subscriptions. Worse,
// `account show` would then pick a default belonging to someone else. Filtering
// on User.Name is what makes the profile answer "what can *this* identity see".
var _ = Describe("Profile reads scoped to an identity", func() {
	const (
		user  = "Brian.Dwyer@broadridge.com"
		admin = "DwyerAdminCld@Broadridge.onmicrosoft.com"
	)

	var ctx context.Context

	// sub builds a profile entry attributed to a specific identity.
	sub := func(id, owner string) cli.Subscription {
		return cli.Subscription{
			EnvironmentName: "AzureCloud",
			ID:              id,
			Name:            "Sub " + id,
			State:           "Enabled",
			TenantID:        "tenant-" + id,
			User:            &cli.User{Name: owner, Type: "user"},
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		useTempCredDir()

		// A profile that already holds both identities' subscriptions, which is
		// the state that produced the bug.
		p, err := profilePath()
		Expect(err).NotTo(HaveOccurred())
		Expect(WriteProfile(cli.Profile{
			InstallationID: uuid.NewString(),
			Subscriptions: []cli.Subscription{
				sub("sub-user", user),
				sub("sub-admin", admin),
			},
		}, p)).To(Succeed())
	})

	It("returns only the hinted identity's subscriptions", func() {
		got, err := ListSubscriptionsCLI(ctx, false, user)
		Expect(err).NotTo(HaveOccurred())

		Expect(got).To(HaveLen(1))
		Expect(got[0].ID).To(Equal("sub-user"))
	})

	It("returns only the other identity's subscriptions for the other hint", func() {
		got, err := ListSubscriptionsCLI(ctx, false, admin)
		Expect(err).NotTo(HaveOccurred())

		Expect(got).To(HaveLen(1))
		Expect(got[0].ID).To(Equal("sub-admin"))
	})

	It("returns the whole profile when no hint narrows it", func() {
		got, err := ListSubscriptionsCLI(ctx, false, "")
		Expect(err).NotTo(HaveOccurred())

		Expect(got).To(HaveLen(2))
	})

	It("leaves the other identity's entries on disk when filtering", func() {
		_, err := ListSubscriptionsCLI(ctx, false, user)
		Expect(err).NotTo(HaveOccurred())

		// Filtering is a read-side concern. Narrowing the view must never
		// delete the identity that was filtered out, or the next invocation as
		// that identity would silently re-enumerate from nothing.
		onDisk, err := LoadProfile()
		Expect(err).NotTo(HaveOccurred())
		Expect(onDisk.Subscriptions).To(HaveLen(2))
	})
})
