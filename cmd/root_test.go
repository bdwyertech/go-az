package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

// newHintCommand mirrors the root command's flag registration so a spec can
// exercise accountHint without mutating the package level rootCmd.
func newHintCommand() *cobra.Command {
	c := &cobra.Command{Use: "spec"}
	c.Flags().String("preferred-username", "", "")
	return c
}

var _ = Describe("--preferred-username", func() {
	BeforeEach(func() {
		for _, k := range []string{"GO_AZ_USERNAME", "AZURE_USERNAME"} {
			prev, had := os.LookupEnv(k)
			Expect(os.Unsetenv(k)).To(Succeed())
			key := k
			DeferCleanup(func() {
				if had {
					Expect(os.Setenv(key, prev)).To(Succeed())
					return
				}
				Expect(os.Unsetenv(key)).To(Succeed())
			})
		}
	})

	It("is registered on the root command without a shorthand", func() {
		f := rootCmd.PersistentFlags().Lookup("preferred-username")
		Expect(f).NotTo(BeNil())
		Expect(f.Shorthand).To(BeEmpty())
	})

	It("resolves the flag value when set", func() {
		c := newHintCommand()
		Expect(c.Flags().Set("preferred-username", "flag@example.com")).To(Succeed())
		Expect(accountHint(c)).To(Equal("flag@example.com"))
	})

	It("prefers the flag over the environment", func() {
		Expect(os.Setenv("GO_AZ_USERNAME", "env@example.com")).To(Succeed())
		c := newHintCommand()
		Expect(c.Flags().Set("preferred-username", "flag@example.com")).To(Succeed())
		Expect(accountHint(c)).To(Equal("flag@example.com"))
	})

	It("leaves a sibling command's hint untouched", func() {
		set := newHintCommand()
		Expect(set.Flags().Set("preferred-username", "flag@example.com")).To(Succeed())
		Expect(accountHint(set)).To(Equal("flag@example.com"))

		sibling := newHintCommand()
		Expect(accountHint(sibling)).To(BeEmpty())
	})
})
