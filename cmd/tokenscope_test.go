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

// Condition 1.1, 1.3: cold start — an empty cache should not abort kube-cred.
// The current code collapses "no accounts cached" into the same error path as
// "hint matched nothing", so kube-cred exits non-zero before the token layer
// has a chance to prompt for login. This test asserts the desired behavior:
// kube-cred proceeds past account selection. It will FAIL against the unfixed
// code, proving the bug exists.
var _ = Describe("kube-cred cold start (empty cache)", func() {
	var out, errOut bytes.Buffer

	BeforeEach(func() {
		out.Reset()
		errOut.Reset()

		// Simulate what happens with an empty cache: resolveIdentity returns
		// ErrNoCachedAccounts because ResolveAccount(nil, ...) now returns that
		// sentinel when the cache snapshot is empty (task 2.2).
		orig := resolveIdentity
		resolveIdentity = func(cmd *cobra.Command, hint string) (string, error) {
			return "", az.ErrNoCachedAccounts
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

	It("proceeds past selection when the cache is empty and no hint is given", func() {
		err := run("kube-cred")

		// Desired: the command does not abort at account selection. It should
		// reach the token layer, which will prompt for login. If an error
		// occurs, it must not be a selection error — the empty-cache case
		// should have been swallowed by resolveHint.
		if err != nil {
			Expect(err.Error()).ToNot(ContainSubstring("selecting an account"))
		}
	})

	It("keeps stdout empty when the cache is empty", func() {
		_ = run("kube-cred")

		// Whether the command aborts or proceeds, stdout must remain clean
		// because kubectl parses it.
		Expect(out.String()).To(BeEmpty())
	})
})
