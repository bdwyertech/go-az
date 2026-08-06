package az

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/google/uuid"
)

var _ = Describe("profile persistence", func() {
	BeforeEach(func() { useTempCredDir() })

	It("writes the profile atomically with owner-only permissions", func() {
		defer setUmask(setUmask(0777))
		p, err := profilePath()
		Expect(err).NotTo(HaveOccurred())

		prof := cli.Profile{InstallationID: uuid.NewString()}
		Expect(WriteProfile(prof, p)).To(Succeed())

		fi, err := os.Stat(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode().Perm()).To(Equal(os.FileMode(0600)))
	})

	It("propagates write failures instead of swallowing them", func() {
		p, err := profilePath()
		Expect(err).NotTo(HaveOccurred())
		// A directory at the destination makes the rename fail.
		Expect(os.MkdirAll(p, 0700)).To(Succeed())
		Expect(WriteProfile(cli.Profile{}, p)).To(MatchError(ContainSubstring("error writing Profile")))
	})

	It("serializes concurrent read-modify-write cycles under one lock", func() {
		p, err := profilePath()
		Expect(err).NotTo(HaveOccurred())
		lock := p + ".lock"

		const workers = 8
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				defer GinkgoRecover()
				for i := 0; i < 10; i++ {
					Expect(withExclusiveLock(context.Background(), lock, func() error {
						prof, _ := cli.LoadProfile(p)
						if prof.InstallationID == "" {
							prof.InstallationID = uuid.NewString()
						}
						prof.Subscriptions = append(prof.Subscriptions, cli.Subscription{ID: uuid.NewString()})
						return WriteProfile(prof, p)
					})).To(Succeed())
				}
			}(w)
		}
		wg.Wait()

		b, err := os.ReadFile(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(json.Valid(b)).To(BeTrue(), "profile must never be observed torn")

		var got cli.Profile
		Expect(json.Unmarshal(b, &got)).To(Succeed())
		Expect(got.Subscriptions).To(HaveLen(workers*10), "no update may be lost")
	})

	It("resolves the profile inside the overridden credential directory", func() {
		p, err := profilePath()
		Expect(err).NotTo(HaveOccurred())
		Expect(filepath.Dir(p)).To(Equal(credDirOverride))
		Expect(filepath.Base(p)).To(Equal("azureProfile.json"))
	})
})
