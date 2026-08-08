package cmd

import (
	"github.com/spf13/cobra"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Every leaf command must report failure by returning an error. A command that
// only sets Run has no way to signal failure: log.Fatal and os.Exit terminate
// the process from inside the handler, which skips deferred cleanup, discards
// released file locks, and makes the failure untestable.
var _ = Describe("Command error handling", func() {
	It("uses RunE on every leaf command", func() {
		var offenders []string
		var walk func(c *cobra.Command)
		walk = func(c *cobra.Command) {
			for _, sub := range c.Commands() {
				walk(sub)
			}
			if len(c.Commands()) > 0 {
				return
			}
			// Cobra generates its own help command the first time the tree is
			// executed. It is framework-owned and cannot fail, so it is not
			// ours to audit.
			if c.Name() == "help" && c.Parent() == c.Root() {
				return
			}
			if c.Run != nil || c.RunE == nil {
				offenders = append(offenders, c.CommandPath())
			}
		}
		walk(rootCmd)
		Expect(offenders).To(BeEmpty())
	})

	It("does not silence errors on the root command", func() {
		Expect(rootCmd.SilenceErrors).To(BeFalse())
	})
})
