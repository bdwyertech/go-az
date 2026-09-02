package az

import (
	"context"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("interactive auth gate", func() {
	BeforeEach(func() { useTempCredDir() })

	It("guards interactive prompts with a lock distinct from the cache lock", func() {
		lp, err := interactiveLockPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Base(lp)).To(Equal("interactive_auth.lock"))
		Expect(lp).NotTo(Equal((&Cache{path: cachePath()}).lockPath()))
	})

	It("retains proxy support on both transports", func() {
		Expect(silentTransport().Proxy).NotTo(BeNil())
		Expect(interactiveTransport().Proxy).NotTo(BeNil())
	})

	It("builds a fresh transport per call so none is shared or mutated", func() {
		Expect(interactiveTransport()).NotTo(BeIdenticalTo(interactiveTransport()))
		Expect(silentTransport()).NotTo(BeIdenticalTo(silentTransport()))
	})

	It("pools connections only on the silent path", func() {
		Expect(silentTransport().DisableKeepAlives).To(BeFalse())
		Expect(interactiveTransport().DisableKeepAlives).To(BeTrue())
	})

	It("bounds a blocked interactive acquisition by the caller context", func() {
		lp, err := interactiveLockPath()
		Expect(err).NotTo(HaveOccurred())

		held := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			Expect(withExclusiveLock(context.Background(), lp, func() error {
				close(held)
				time.Sleep(750 * time.Millisecond)
				return nil
			})).To(Succeed())
		}()
		<-held

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		ran := false
		err = withExclusiveLock(ctx, lp, func() error { ran = true; return nil })
		Expect(err).To(MatchError(context.DeadlineExceeded))
		Expect(ran).To(BeFalse())
		<-done
	})
})
