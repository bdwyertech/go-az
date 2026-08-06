package az

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

var _ = Describe("regression prevention", func() {
	var c *Cache

	BeforeEach(func() {
		useTempCredDir()
		c = &Cache{path: cachePath()}
	})

	It("keeps the on-disk cache byte-compatible with the Azure CLI format", func() {
		// MSAL hands Export compact JSON; the Azure CLI writes it 2-space
		// indented. Export must preserve that framing exactly.
		compact := []byte(`{"Account":{"k":{"username":"u@example.com"}},"AccessToken":{}}`)
		Expect(c.Export(context.Background(), fakeMarshaler{compact}, cache.ExportHints{})).To(Succeed())

		onDisk, err := os.ReadFile(c.path)
		Expect(err).NotTo(HaveOccurred())

		// json.Indent reframes without reordering keys; MarshalIndent would
		// sort them and silently change the file the Azure CLI reads back.
		expected := new(bytes.Buffer)
		Expect(json.Indent(expected, compact, "", "  ")).To(Succeed())
		Expect(string(onDisk)).To(Equal(expected.String()))
		Expect(onDisk).To(MatchJSON(compact))
	})

	It("leaves no temporary or stray files beside the cache", func() {
		Expect(c.Export(context.Background(), fakeMarshaler{[]byte(`{}`)}, cache.ExportHints{})).To(Succeed())
		entries, err := os.ReadDir(credDirOverride)
		Expect(err).NotTo(HaveOccurred())
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		Expect(names).To(ConsistOf("go_msal_token_cache.json", "go_msal_token_cache.json.lock"))
	})

	It("does not make an uncontended single-process round trip wait", func() {
		// A lone process must never block: 200 lock cycles have to complete
		// well inside any retry/backoff window.
		start := time.Now()
		for i := 0; i < 200; i++ {
			Expect(c.Export(context.Background(), fakeMarshaler{[]byte(`{"n":1}`)}, cache.ExportHints{})).To(Succeed())
			Expect(c.Replace(context.Background(), &fakeUnmarshaler{}, cache.ReplaceHints{})).To(Succeed())
		}
		Expect(time.Since(start)).To(BeNumerically("<", 2*time.Second))
	})

	It("re-reads a cache rewritten by another process", func() {
		first := []byte(`{"Account":{"a":{"username":"first@example.com"}}}`)
		Expect(c.Export(context.Background(), fakeMarshaler{first}, cache.ExportHints{})).To(Succeed())

		// Simulate the Azure CLI replacing the file underneath us.
		second := []byte(`{"Account":{"b":{"username":"second@example.com"}}}`)
		Expect(WriteFileAtomic(c.path, second, credFileMode)).To(Succeed())

		u := &fakeUnmarshaler{}
		Expect(c.Replace(context.Background(), u, cache.ReplaceHints{})).To(Succeed())
		Expect(u.seen).To(MatchJSON(second))
	})
})
