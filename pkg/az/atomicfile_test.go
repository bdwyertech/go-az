package az

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WriteFileAtomic", func() {
	var dir, path string

	BeforeEach(func() {
		dir = useTempCredDir()
		path = filepath.Join(dir, "target.json")
	})

	It("never exposes a torn or empty file to a concurrent reader", func() {
		old := []byte(strings.Repeat("a", 4096))
		Expect(WriteFileAtomic(path, old, 0600)).To(Succeed())
		new := []byte(strings.Repeat("b", 4096))

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				Expect(WriteFileAtomic(path, new, 0600)).To(Succeed())
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				got, err := os.ReadFile(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(got)).To(Equal(4096))
			}
		}()
		wg.Wait()
	})

	It("stages the temp file in the target directory so the rename is same-filesystem", func() {
		Expect(WriteFileAtomic(path, []byte("x"), 0600)).To(Succeed())
		entries, err := os.ReadDir(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].Name()).To(Equal("target.json"))
	})

	It("leaves the original intact and no stray temp file when the write fails", func() {
		Expect(WriteFileAtomic(path, []byte("original"), 0600)).To(Succeed())

		// A directory as the target makes the rename fail.
		bad := filepath.Join(dir, "sub")
		Expect(os.Mkdir(bad, 0700)).To(Succeed())
		Expect(WriteFileAtomic(bad, []byte("nope"), 0600)).NotTo(Succeed())

		got, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(got)).To(Equal("original"))

		entries, err := os.ReadDir(dir)
		Expect(err).NotTo(HaveOccurred())
		for _, e := range entries {
			Expect(e.Name()).NotTo(HavePrefix("."))
		}
	})

	It("applies the requested mode exactly, unaffected by umask", func() {
		prev := setUmask(0777)
		defer setUmask(prev)

		Expect(WriteFileAtomic(path, []byte("x"), 0600)).To(Succeed())
		info, err := os.Stat(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
	})
})
