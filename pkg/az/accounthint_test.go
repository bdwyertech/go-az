package az

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Account hint resolution", func() {
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

	It("is empty when neither the flag nor the environment supplies one", func() {
		Expect(ResolveAccountHint("")).To(BeEmpty())
	})

	It("prefers the flag over both environment variables", func() {
		Expect(os.Setenv("GO_AZ_USERNAME", "env@example.com")).To(Succeed())
		Expect(os.Setenv("AZURE_USERNAME", "azure@example.com")).To(Succeed())
		Expect(ResolveAccountHint("flag@example.com")).To(Equal("flag@example.com"))
	})

	It("prefers GO_AZ_USERNAME over AZURE_USERNAME", func() {
		Expect(os.Setenv("GO_AZ_USERNAME", "goaz@example.com")).To(Succeed())
		Expect(os.Setenv("AZURE_USERNAME", "azure@example.com")).To(Succeed())
		Expect(ResolveAccountHint("")).To(Equal("goaz@example.com"))
	})

	It("falls back to AZURE_USERNAME when GO_AZ_USERNAME is empty", func() {
		Expect(os.Setenv("GO_AZ_USERNAME", "")).To(Succeed())
		Expect(os.Setenv("AZURE_USERNAME", "azure@example.com")).To(Succeed())
		Expect(ResolveAccountHint("")).To(Equal("azure@example.com"))
	})

	It("trims surrounding whitespace from every source", func() {
		Expect(ResolveAccountHint("  flag@example.com  ")).To(Equal("flag@example.com"))

		Expect(os.Setenv("GO_AZ_USERNAME", "  goaz@example.com\n")).To(Succeed())
		Expect(ResolveAccountHint("")).To(Equal("goaz@example.com"))
	})

	It("ignores a whitespace-only environment value", func() {
		Expect(os.Setenv("GO_AZ_USERNAME", "   ")).To(Succeed())
		Expect(os.Setenv("AZURE_USERNAME", "azure@example.com")).To(Succeed())
		Expect(ResolveAccountHint("")).To(Equal("azure@example.com"))
	})
})
