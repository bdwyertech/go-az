package az

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// fakeMarshaler feeds Export a fixed JSON payload.
type fakeMarshaler struct{ payload []byte }

func (f fakeMarshaler) Marshal() ([]byte, error) { return f.payload, nil }

// fakeUnmarshaler records what Replace handed back.
type fakeUnmarshaler struct {
	mu   sync.Mutex
	seen []byte
}

func (f *fakeUnmarshaler) Unmarshal(b []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seen = append([]byte(nil), b...)
	return nil
}

var _ = Describe("Cache", func() {
	var dir string
	var c *Cache

	BeforeEach(func() {
		dir = useTempCredDir()
		c = &Cache{path: cachePath()}
	})

	It("exposes no mutable control-flow field", func() {
		t := reflect.TypeOf(Cache{})
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			Expect(f.Type.Kind()).NotTo(Equal(reflect.Bool),
				"field %q is a bool; control flow must not hinge on cache state", f.Name)
			Expect(f.Name).NotTo(Equal("locked"))
			Expect(f.Name).NotTo(Equal("mutex"))
		}
	})

	It("writes the cache file with owner-only permissions", func() {
		defer setUmask(setUmask(0777))
		payload := []byte(`{"Account":{}}`)
		Expect(c.Export(context.Background(), fakeMarshaler{payload}, cache.ExportHints{})).To(Succeed())

		fi, err := os.Stat(c.path)
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode().Perm()).To(Equal(os.FileMode(0600)))

		di, err := os.Stat(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(di.Mode().Perm()).To(Equal(os.FileMode(0700)))
	})

	It("round-trips an exported payload through Replace", func() {
		payload := []byte(`{"Account":{"a":{"username":"u@example.com"}}}`)
		Expect(c.Export(context.Background(), fakeMarshaler{payload}, cache.ExportHints{})).To(Succeed())

		u := &fakeUnmarshaler{}
		other := &Cache{path: c.path}
		Expect(other.Replace(context.Background(), u, cache.ReplaceHints{})).To(Succeed())
		Expect(json.Valid(u.seen)).To(BeTrue())
		Expect(u.seen).To(MatchJSON(payload))
	})

	It("treats a missing cache file as an empty cache", func() {
		u := &fakeUnmarshaler{}
		Expect(c.Replace(context.Background(), u, cache.ReplaceHints{})).To(Succeed())
		Expect(u.seen).To(BeEmpty())
	})

	It("survives concurrent Export and Replace without racing", func() {
		const workers = 8
		const iterations = 25
		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				defer GinkgoRecover()
				payload := []byte(`{"Account":{"a":{"username":"u@example.com"}}}`)
				for i := 0; i < iterations; i++ {
					Expect(c.Export(context.Background(), fakeMarshaler{payload}, cache.ExportHints{})).To(Succeed())
					Expect(c.Replace(context.Background(), &fakeUnmarshaler{}, cache.ReplaceHints{})).To(Succeed())
				}
			}(w)
		}
		wg.Wait()

		b, err := os.ReadFile(c.path)
		Expect(err).NotTo(HaveOccurred())
		Expect(json.Valid(b)).To(BeTrue(), "cache file must never be observed torn")
	})
})
