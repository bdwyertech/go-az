package az

import (
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const sampleCreds = `{
  "Account": {
    "k1": {
      "home_account_id": "hid-1",
      "realm": "organizations",
      "username": "User@Example.com"
    },
    "k2": {
      "home_account_id": "hid-2",
      "realm": "11111111-1111-1111-1111-111111111111",
      "username": "other@example.com"
    }
  },
  "IdToken": {
    "t1": { "home_account_id": "hid-1", "secret": "assertion-1" },
    "t2": { "home_account_id": "hid-2", "secret": "assertion-2" }
  },
  "RefreshToken": {
    "r1": { "home_account_id": "hid-1", "secret": "refresh-1" }
  }
}`

var _ = Describe("credential read paths", func() {
	BeforeEach(func() {
		useTempCredDir()
		Expect(os.WriteFile(cachePath(), []byte(sampleCreds), credFileMode)).To(Succeed())
	})

	It("matches a username case-insensitively", func() {
		creds := LoadLocalCreds()
		Expect(creds.AssertionForUser("user@example.com")).To(Equal("assertion-1"))
		Expect(creds.AssertionForUser("USER@EXAMPLE.COM")).To(Equal("assertion-1"))
		Expect(creds.AssertionForUser("nobody@example.com")).To(BeEmpty())
	})

	It("returns a credential from First", func() {
		Expect(LoadLocalCreds().First()).NotTo(BeNil())
	})

	It("returns an empty struct when no cache file exists", func() {
		Expect(os.Remove(cachePath())).To(Succeed())
		Expect(LoadLocalCreds().Account).To(BeEmpty())
	})

	It("serves concurrent readers without racing", func() {
		var wg sync.WaitGroup
		for w := 0; w < 8; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				for i := 0; i < 20; i++ {
					creds := LoadLocalCreds()
					Expect(creds.First()).NotTo(BeNil())
					Expect(creds.AssertionForUser("user@example.com")).To(Equal("assertion-1"))
				}
			}()
		}
		wg.Wait()
	})
})
