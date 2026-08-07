package az

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("State File", func() {
	var dir string

	BeforeEach(func() {
		dir = useTempCredDir()
	})

	It("resolves beside the token cache in the temporary credential directory", func() {
		p, err := statePath()
		Expect(err).NotTo(HaveOccurred())
		Expect(p).To(Equal(filepath.Join(dir, "go_az_state.json")))
	})

	It("treats a missing state file as the zero value", func() {
		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(State{}))
	})

	It("round trips a stored state", func() {
		want := State{ActiveUsername: "user@example.com", ActiveHomeAccountID: "oid-1.tenant-a"}
		Expect(StoreState(context.Background(), want)).To(Succeed())

		got, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(want))
	})

	It("writes the state file readable only by its owner", func() {
		Expect(StoreState(context.Background(), State{ActiveUsername: "a@b.c"})).To(Succeed())

		p, err := statePath()
		Expect(err).NotTo(HaveOccurred())
		fi, err := os.Stat(p)
		Expect(err).NotTo(HaveOccurred())
		Expect(fi.Mode().Perm()).To(Equal(os.FileMode(0600)))
	})

	It("treats a corrupt state file as the zero value", func() {
		p, err := statePath()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(p, []byte("{not json"), 0600)).To(Succeed())

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(State{}))
	})

	It("overwrites a previously stored state", func() {
		Expect(StoreState(context.Background(), State{ActiveUsername: "first@example.com"})).To(Succeed())
		Expect(StoreState(context.Background(), State{ActiveUsername: "second@example.com"})).To(Succeed())

		s, err := LoadState(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(s.ActiveUsername).To(Equal("second@example.com"))
	})

	It("keeps the active account out of the Azure CLI profile", func() {
		Expect(StoreState(context.Background(), State{ActiveUsername: "user@example.com"})).To(Succeed())

		p, err := profilePath()
		Expect(err).NotTo(HaveOccurred())
		Expect(WriteProfile(cli.Profile{InstallationID: "id"}, p)).To(Succeed())

		b, err := os.ReadFile(p)
		Expect(err).NotTo(HaveOccurred())

		var doc map[string]any
		Expect(json.Unmarshal(b, &doc)).To(Succeed())
		Expect(doc).NotTo(HaveKey("activeUsername"))
		Expect(doc).NotTo(HaveKey("activeHomeAccountId"))
	})
})
