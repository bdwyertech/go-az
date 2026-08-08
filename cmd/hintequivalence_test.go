package cmd

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment fallback is equivalent to the flag (Property 2)", func() {
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

	// The hint is the only input the enumeration path takes, so proving the two
	// spellings produce the same hint proves they produce the same credentials.
	It("yields the same hint from GO_AZ_USERNAME as from the flag", func() {
		const want = "Brian.Dwyer@broadridge.com"

		viaFlag := newHintCommand()
		Expect(viaFlag.Flags().Set("preferred-username", want)).To(Succeed())

		Expect(os.Setenv("GO_AZ_USERNAME", want)).To(Succeed())
		viaEnv := newHintCommand()

		Expect(accountHint(viaEnv)).To(Equal(accountHint(viaFlag)))
		Expect(accountHint(viaEnv)).To(Equal(want))
	})

	It("yields the same hint from AZURE_USERNAME as from the flag", func() {
		const want = "DwyerAdminCld@Broadridge.onmicrosoft.com"

		viaFlag := newHintCommand()
		Expect(viaFlag.Flags().Set("preferred-username", want)).To(Succeed())

		Expect(os.Setenv("AZURE_USERNAME", want)).To(Succeed())
		viaEnv := newHintCommand()

		Expect(accountHint(viaEnv)).To(Equal(accountHint(viaFlag)))
	})
})
