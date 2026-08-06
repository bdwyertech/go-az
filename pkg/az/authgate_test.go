package az

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	It("never mutates a transport after handing it to MSAL", func() {
		src, err := os.ReadFile("authgate.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(src)).NotTo(ContainSubstring("DisableKeepAlives ="))
		auth, err := os.ReadFile("auth.go")
		Expect(err).NotTo(HaveOccurred())
		Expect(string(auth)).NotTo(ContainSubstring("DisableKeepAlives"))
	})

	It("acquires the interactive gate linearly, without recursing", func() {
		auth, err := os.ReadFile("auth.go")
		Expect(err).NotTo(HaveOccurred())
		body := string(auth)
		start := strings.Index(body, "func GetToken(")
		Expect(start).To(BeNumerically(">", 0))
		end := strings.Index(body[start:], "\nfunc ")
		Expect(end).To(BeNumerically(">", 0))
		Expect(body[start+len("func GetToken("):start+end]).
			NotTo(ContainSubstring("GetToken(ctx"), "GetToken must not call itself")
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
