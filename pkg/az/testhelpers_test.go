package az

import (
	"os"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// useTempCredDir redirects the credential directory to a fresh temporary
// directory for the duration of the current spec and rebuilds the package
// level credCache so it points at the temporary location. The previous values
// are restored during spec cleanup.
func useTempCredDir() string {
	dir, err := os.MkdirTemp("", "go-az-spec-")
	Expect(err).NotTo(HaveOccurred())
	Expect(os.Chmod(dir, 0700)).To(Succeed())

	prevDir := credDirOverride
	prevCache := credCache

	credDirOverride = dir
	credCache = &Cache{path: cachePath()}

	DeferCleanup(func() {
		credDirOverride = prevDir
		credCache = prevCache
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	return dir
}

// twoIdentityAccounts is a hermetic, two-identity token cache fixture: a
// regular user recorded as the Active Account, and an admin account that is
// not. Callers combine this with useTempCredDir and a fake Enumerator so a
// spec never depends on a real cached login.
func twoIdentityAccounts() (user, admin public.Account) {
	user = acct("Brian.Dwyer@broadridge.com", "oid-user", "tenant-a")
	admin = acct("DwyerAdminCld@Broadridge.onmicrosoft.com", "oid-admin", "tenant-b")
	return
}
