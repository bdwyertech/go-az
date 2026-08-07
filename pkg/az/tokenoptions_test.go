package az

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Token option immutability", func() {
	Describe("withScope", func() {
		It("leaves the caller's slice untouched", func() {
			// Spare capacity is what makes the bug reachable: append writes into
			// the caller's array instead of allocating.
			caller := make([]string, 1, 4)
			caller[0] = "https://management.azure.com/.default"

			got := withScope(caller, "https://vault.azure.net")

			Expect(caller).To(HaveLen(1))
			Expect(caller[0]).To(Equal("https://management.azure.com/.default"))
			Expect(got).To(HaveLen(2))
			Expect(got[1]).To(Equal("https://vault.azure.net/.default"))
		})

		It("does not share a backing array with the caller", func() {
			caller := []string{"a"}
			got := withScope(caller, "")
			got[0] = "mutated"
			Expect(caller[0]).To(Equal("a"))
		})

		It("emits each scope exactly once", func() {
			got := withScope([]string{"a", "b", "a"}, "")
			Expect(got).To(Equal([]string{"a", "b"}))
		})

		It("does not repeat a resource scope the caller already supplied", func() {
			got := withScope([]string{"b/.default"}, "b")
			Expect(got).To(Equal([]string{"b/.default"}))
		})

		It("returns nil for no scopes and no resource", func() {
			Expect(withScope(nil, "")).To(BeNil())
		})

		It("appends the resource default scope when there are no scopes", func() {
			Expect(withScope(nil, "https://vault.azure.net")).
				To(Equal([]string{"https://vault.azure.net/.default"}))
		})
	})

	Describe("GetAccessToken", func() {
		It("does not mutate the caller's AccessTokenOptions", func() {
			useTempCredDir()
			scope := make([]string, 1, 4)
			scope[0] = "https://management.azure.com/.default"
			opts := AccessTokenOptions{Scope: scope, Resource: "https://vault.azure.net"}
			before := opts

			// The call fails without a cached credential; the point is that the
			// caller's struct and slice survive the attempt unchanged.
			// A cancelled context keeps the attempt from ever prompting.
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, _ = GetAccessToken(ctx, opts)

			Expect(opts).To(Equal(before))
			Expect(opts.Scope).To(HaveLen(1))
			Expect(opts.Scope[0]).To(Equal("https://management.azure.com/.default"))
		})
	})
})
