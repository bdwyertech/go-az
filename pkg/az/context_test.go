package az

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Context propagation", func() {
	var dir string

	BeforeEach(func() {
		dir = useTempCredDir()
	})

	Describe("locking", func() {
		It("returns the context error instead of blocking forever", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := withExclusiveLock(ctx, filepath.Join(dir, "x.lock"), func() error {
				Fail("the guarded function must not run under a cancelled context")
				return nil
			})

			Expect(err).To(MatchError(context.Canceled))
		})

		It("releases the lock so a later acquisition succeeds", func() {
			path := filepath.Join(dir, "y.lock")
			ctx := context.Background()

			Expect(withExclusiveLock(ctx, path, func() error { return nil })).To(Succeed())
			Expect(withExclusiveLock(ctx, path, func() error { return nil })).To(Succeed())
		})

		It("releases the lock even when the guarded function fails", func() {
			path := filepath.Join(dir, "z.lock")
			ctx := context.Background()
			boom := os.ErrInvalid

			Expect(withExclusiveLock(ctx, path, func() error { return boom })).To(MatchError(boom))
			// A leaked lock would make this second acquisition hang.
			Expect(withExclusiveLock(ctx, path, func() error { return nil })).To(Succeed())
		})
	})

	Describe("enumeration", func() {
		It("returns errors rather than ending the process", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// Before this change these paths called log.Fatal, which no caller
			// and no spec could recover from.
			_, err := ListTenants(ctx)
			Expect(err).To(HaveOccurred())

			_, err = ListSubscriptionsForTenant(ctx, "t1")
			Expect(err).To(HaveOccurred())
		})

		It("surfaces a cancelled context from the profile writer", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			Expect(BuildProfile(ctx)).To(MatchError(context.Canceled))
		})
	})
})
