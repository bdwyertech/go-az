package az

import (
	"context"
	"fmt"
	"net/http"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// msalAccountJSON returns a minimal MSAL-format cache JSON seeding the given
// accounts. The key format matches what MSAL writes:
// "{home_account_id}-{environment}-{realm}".
func msalAccountJSON(accounts ...public.Account) string {
	entries := ""
	for i, a := range accounts {
		if i > 0 {
			entries += ","
		}
		key := fmt.Sprintf("%s-%s-%s", a.HomeAccountID, a.Environment, a.Realm)
		entries += fmt.Sprintf(
			`%q:{"home_account_id":%q,"environment":%q,"realm":%q,"local_account_id":%q,"username":%q,"authority_type":"MSSTS"}`,
			key, a.HomeAccountID, a.Environment, a.Realm, a.LocalAccountID, a.PreferredUsername,
		)
	}
	return fmt.Sprintf(`{"Account":{%s}}`, entries)
}

// seedMSALCache writes a minimal MSAL token cache containing the given
// accounts into the temp credential directory managed by useTempCredDir.
func seedMSALCache(accounts ...public.Account) {
	ExpectWithOffset(1, os.WriteFile(cachePath(), []byte(msalAccountJSON(accounts...)), credFileMode)).To(Succeed())
}

// failingTransport is an http.RoundTripper that fails the test if any request
// is made. Used to verify hint derivation is network-free.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	Fail("unexpected HTTP request during hint derivation")
	return nil, nil // unreachable
}

var _ = Describe("cachedLoginHint", func() {
	var ctx context.Context

	BeforeEach(func() {
		useTempCredDir()
		ctx = context.Background()
	})

	// buildPubClient creates a public.Client tied to the temp credential cache
	// and optionally using a custom transport. Passing nil for the transport
	// uses the default.
	buildPubClient := func(t http.RoundTripper) public.Client {
		opts := []public.Option{public.WithCache(credCache)}
		if t != nil {
			opts = append(opts, public.WithHTTPClient(&http.Client{Transport: t}))
		}
		c, err := public.New(AZ_CLIENT_ID,
			opts...,
		)
		Expect(err).NotTo(HaveOccurred())
		return c
	}

	It("returns the Account Hint username when an explicit hint matches a cached account", func() {
		user := acct("user@contoso.com", "oid-1", "tenant-a")
		admin := acct("admin@contoso.com", "oid-2", "tenant-b")
		seedMSALCache(user, admin)

		pc := buildPubClient(nil)
		hint := cachedLoginHint(ctx, pc, TokenOptions{
			PreferredUsername: "user@contoso.com",
		})
		Expect(hint).To(Equal("user@contoso.com"))
	})

	It("returns the sole cached account's username when no hint is supplied", func() {
		user := acct("solo@contoso.com", "oid-solo", "tenant-a")
		seedMSALCache(user)

		pc := buildPubClient(nil)
		hint := cachedLoginHint(ctx, pc, TokenOptions{})
		Expect(hint).To(Equal("solo@contoso.com"))
	})

	It("returns empty string when the cache holds no accounts", func() {
		// No seedMSALCache call — empty credential directory.
		pc := buildPubClient(nil)
		hint := cachedLoginHint(ctx, pc, TokenOptions{})
		Expect(hint).To(BeEmpty())
	})

	It("returns empty string on ambiguous resolution rather than an error", func() {
		user := acct("user@contoso.com", "oid-1", "tenant-a")
		admin := acct("admin@contoso.com", "oid-2", "tenant-b")
		seedMSALCache(user, admin)
		// No active account set, no hint — two candidates are ambiguous.

		pc := buildPubClient(nil)
		hint := cachedLoginHint(ctx, pc, TokenOptions{})
		Expect(hint).To(BeEmpty())
	})

	It("returns empty string when the hint matches no cached account", func() {
		user := acct("user@contoso.com", "oid-1", "tenant-a")
		seedMSALCache(user)

		pc := buildPubClient(nil)
		hint := cachedLoginHint(ctx, pc, TokenOptions{
			PreferredUsername: "nobody@example.com",
		})
		Expect(hint).To(BeEmpty())
	})

	It("returns the active account's username when no hint is given and multiple accounts exist", func() {
		user := acct("user@contoso.com", "oid-1", "tenant-a")
		admin := acct("admin@contoso.com", "oid-2", "tenant-b")
		seedMSALCache(user, admin)

		Expect(StoreState(ctx, State{
			ActiveUsername:      admin.PreferredUsername,
			ActiveHomeAccountID: admin.HomeAccountID,
		})).To(Succeed())

		pc := buildPubClient(nil)
		hint := cachedLoginHint(ctx, pc, TokenOptions{})
		Expect(hint).To(Equal("admin@contoso.com"))
	})

	It("issues no HTTP request while deriving a hint", func() {
		user := acct("user@contoso.com", "oid-1", "tenant-a")
		seedMSALCache(user)

		pc := buildPubClient(failingTransport{})
		hint := cachedLoginHint(ctx, pc, TokenOptions{})
		Expect(hint).To(Equal("user@contoso.com"))
	})

	It("issues no HTTP request even when the cache is empty", func() {
		pc := buildPubClient(failingTransport{})
		hint := cachedLoginHint(ctx, pc, TokenOptions{})
		Expect(hint).To(BeEmpty())
	})
})
