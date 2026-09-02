package cmd

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Attribution matches the acquiring identity", func() {
	It("writes the resolved username to the error stream, not the output stream", func() {
		var out, errOut bytes.Buffer

		c := newHintCommand()
		c.SetOut(&out)
		c.SetErr(&errOut)

		emitAttribution(c, "user@contoso.com")

		Expect(errOut.String()).To(ContainSubstring("user@contoso.com"))
		Expect(out.String()).To(BeEmpty())
	})

	It("leaves a JSON array on stdout parseable as an array", func() {
		var out, errOut bytes.Buffer

		c := newHintCommand()
		c.SetOut(&out)
		c.SetErr(&errOut)

		emitAttribution(c, "user@contoso.com")

		// Attribution must never leak into the machine-readable stream, so the
		// same buffer that carries the payload still decodes as a bare array.
		enc := json.NewEncoder(&out)
		Expect(enc.Encode([]string{"t1", "t2"})).To(Succeed())

		var arr []string
		Expect(json.Unmarshal(out.Bytes(), &arr)).To(Succeed())
		Expect(arr).To(Equal([]string{"t1", "t2"}))
	})

	It("says nothing when no identity was resolved", func() {
		var errOut bytes.Buffer

		c := newHintCommand()
		c.SetErr(&errOut)

		emitAttribution(c, "")

		Expect(errOut.String()).To(BeEmpty())
	})
})
