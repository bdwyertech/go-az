package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/bdwyertech/go-az/pkg/az"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

// fakeDiscovery stands in for the live cache and ARM/Graph calls: it maps each
// identity to the tenants that identity can see, which is exactly the
// distinction the bug erased.
func fakeDiscovery(byUser map[string][]az.Organization) {
	prevResolve, prevList := resolveIdentity, listOrganizations

	resolveIdentity = func(_ *cobra.Command, hint string) (string, error) {
		if hint == "" {
			// No hint means the Active Account, which this fixture records as
			// the admin identity.
			return "DwyerAdminCld@Broadridge.onmicrosoft.com", nil
		}
		if _, ok := byUser[hint]; !ok {
			return "", az.ErrNoMatchingAccount
		}
		return hint, nil
	}
	listOrganizations = func(_ *cobra.Command, username string) ([]az.Organization, error) {
		return byUser[username], nil
	}

	DeferCleanup(func() {
		resolveIdentity, listOrganizations = prevResolve, prevList
	})
}

var _ = Describe("organizations end to end", func() {
	const (
		user  = "Brian.Dwyer@broadridge.com"
		admin = "DwyerAdminCld@Broadridge.onmicrosoft.com"
	)

	var out, errOut bytes.Buffer

	// runOrganizations drives the real command tree from the root, so the spec
	// exercises the same dispatch, flag parsing, and output path a user gets.
	// Executing the leaf directly would walk up to the root anyway and print
	// help instead of running RunE.
	runOrganizations := func(args ...string) error {
		out.Reset()
		errOut.Reset()
		// Cobra flag values persist on the shared command object, so a run
		// without a hint would otherwise inherit the previous spec's.
		_ = rootCmd.PersistentFlags().Set("preferred-username", "")
		rootCmd.SetOut(&out)
		rootCmd.SetErr(&errOut)
		rootCmd.SetArgs(append([]string{"organizations"}, args...))
		return rootCmd.ExecuteContext(context.Background())
	}

	BeforeEach(func() {
		fakeDiscovery(map[string][]az.Organization{
			user:  {{ID: "tenant-a", DisplayName: "Broadridge"}},
			admin: {{ID: "tenant-b", DisplayName: "Broadridge Admin"}},
		})
		DeferCleanup(func() { rootCmd.SetArgs(nil) })
	})

	It("lists the hinted identity's tenants, not the Active Account's", func() {
		Expect(runOrganizations("--preferred-username", user)).To(Succeed())

		Expect(out.String()).To(ContainSubstring("tenant-a"))
		Expect(out.String()).NotTo(ContainSubstring("tenant-b"))
		Expect(errOut.String()).To(ContainSubstring(user))
	})

	It("falls back to the Active Account without a hint", func() {
		Expect(runOrganizations()).To(Succeed())

		Expect(out.String()).To(ContainSubstring("tenant-b"))
		Expect(out.String()).NotTo(ContainSubstring("tenant-a"))
	})

	It("aborts with empty stdout when the hint matches nothing", func() {
		err := runOrganizations("--preferred-username", "nobody@example.com")

		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, az.ErrNoMatchingAccount)).To(BeTrue())
		Expect(out.String()).To(BeEmpty())
	})

	It("emits a bare JSON array on stdout with attribution on stderr", func() {
		Expect(runOrganizations("--preferred-username", user, "--json")).To(Succeed())

		var arr []az.Organization
		Expect(json.Unmarshal(out.Bytes(), &arr)).To(Succeed())
		Expect(arr).To(HaveLen(1))
		Expect(arr[0].ID).To(Equal("tenant-a"))
		Expect(errOut.String()).To(ContainSubstring(user))
	})
})
