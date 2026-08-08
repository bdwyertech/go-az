package cmd

import (
	"bytes"
	"context"
	"errors"

	"github.com/bdwyertech/go-az/pkg/az"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

// Condition 5.4: token-issuing commands abort on an unmatched hint before they
// request anything. `kube-cred` writes to a stream kubectl parses, so a prompt
// or a wrong-identity token is far worse than a non-zero exit.
var _ = Describe("token commands scoped to an identity", func() {
	var out, errOut bytes.Buffer

	// Swap identity resolution for a fake that rejects the hint, and record
	// whether resolution ran at all.
	resolved := false
	BeforeEach(func() {
		out.Reset()
		errOut.Reset()
		resolved = false

		orig := resolveIdentity
		resolveIdentity = func(cmd *cobra.Command, hint string) (string, error) {
			resolved = true
			return "", errors.New("no cached account matches " + hint)
		}
		DeferCleanup(func() {
			resolveIdentity = orig
			rootCmd.SetArgs(nil)
			_ = rootCmd.PersistentFlags().Set("preferred-username", "")
		})
	})

	run := func(args ...string) error {
		rootCmd.SetOut(&out)
		rootCmd.SetErr(&errOut)
		rootCmd.SetArgs(args)
		return rootCmd.ExecuteContext(context.Background())
	}

	It("aborts kube-cred with empty stdout when the hint matches nothing", func() {
		err := run("kube-cred", "--preferred-username", "nobody@example.com")

		Expect(err).To(HaveOccurred())
		Expect(resolved).To(BeTrue())
		Expect(out.String()).To(BeEmpty())
	})

	It("aborts get-access-token with empty stdout when the hint matches nothing", func() {
		err := run("account", "get-access-token", "--preferred-username", "nobody@example.com")

		Expect(err).To(HaveOccurred())
		Expect(resolved).To(BeTrue())
		Expect(out.String()).To(BeEmpty())
	})

	It("reports the account selection rather than a token failure", func() {
		err := run("kube-cred", "--preferred-username", "nobody@example.com")

		// The message has to name the real problem. A token error here would
		// send the user looking at scopes and network instead of their hint.
		Expect(err).To(MatchError(ContainSubstring("selecting an account")))
	})

	It("passes the resolved username, not the raw hint, to the credential", func() {
		// ResolveEnumerationIdentity returns the cache's canonical spelling, so
		// the credential must be built from its result rather than the flag.
		var seen string
		resolveIdentity = func(cmd *cobra.Command, hint string) (string, error) {
			seen = hint
			return "", errors.New("stop before any network call")
		}

		_ = run("kube-cred", "--preferred-username", "Nobody@Example.com")

		Expect(seen).To(Equal("Nobody@Example.com"))
		Expect(az.ResolveAccountHint("")).To(BeEmpty())
	})
})
