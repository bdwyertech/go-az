package az

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/mitchellh/go-homedir"
)

var credCache *Cache

func init() {
	credCache = &Cache{path: cachePath()}
}

// credDirOverride, when non-empty, replaces the default ~/.azure credential
// directory. Tests point this at a per-spec temporary directory so no spec
// ever touches the real user credential store.
var credDirOverride string

func cacheDir() (string, error) {
	d := credDirOverride
	if d == "" {
		home, err := homedir.Dir()
		if err != nil {
			return "", err
		}
		d = filepath.Join(home, ".azure")
	}
	if err := os.MkdirAll(d, credDirMode); err != nil {
		return "", err
	}
	// MkdirAll honours the umask; force the mode so credentials stay private.
	if err := os.Chmod(d, credDirMode); err != nil {
		return "", err
	}
	return d, nil
}

// credDirMode and credFileMode keep the credential directory and its contents
// readable only by the owning user.
const (
	credDirMode  = os.FileMode(0700)
	credFileMode = os.FileMode(0600)
)

func cachePath() string {
	d, err := cacheDir()
	if err != nil {
		log.Error(err)
		return ""
	}
	return filepath.Join(d, "go_msal_token_cache.json")
}

// Cache is the MSAL token cache accessor. All exported state is guarded: mu
// serializes goroutines within this process, and an advisory file lock
// serializes this process against other tools sharing the same file. The Cache
// deliberately exposes no field a caller can mutate to influence control flow.
type Cache struct {
	path string

	mu    sync.Mutex
	bytes []byte
}

// lockPath returns the advisory lock file guarding the token cache.
func (c *Cache) lockPath() string { return c.path + ".lock" }

func (c *Cache) Export(ctx context.Context, m cache.Marshaler, k cache.ExportHints) error {
	jsonBytes, err := m.Marshal()
	if err != nil {
		log.Error(err)
		return nil
	}
	b := new(bytes.Buffer)
	if err = json.Indent(b, jsonBytes, "", "  "); err != nil {
		log.Error(err)
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if bytes.Equal(c.bytes, b.Bytes()) {
		return nil
	}
	err = withExclusiveLock(ctx, c.lockPath(), func() error {
		return WriteFileAtomic(c.path, b.Bytes(), credFileMode)
	})
	if err != nil {
		log.Error(err)
		return nil
	}
	c.bytes = b.Bytes()
	return nil
}

func (c *Cache) Replace(ctx context.Context, u cache.Unmarshaler, k cache.ReplaceHints) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var raw []byte
	err := withSharedLock(ctx, c.lockPath(), func() error {
		b, err := os.ReadFile(c.path)
		if errors.Is(err, os.ErrNotExist) {
			// A missing file is an empty cache, not a failure.
			return nil
		}
		raw = b
		return err
	})
	if err != nil {
		log.Error(err)
		return nil
	}
	if len(raw) == 0 {
		return nil
	}
	c.bytes = raw
	if err = u.Unmarshal(raw); err != nil {
		log.Error(err)
	}
	return nil
}

type LocalCreds struct {
	Account map[string]struct {
		HomeAccountID  string `json:"home_account_id"`
		Environment    string `json:"environment"`
		Realm          string `json:"realm"`
		LocalAccountID string `json:"local_account_id"`
		Username       string `json:"username"`
		AuthorityType  string `json:"authority_type"`
		AccountSource  string `json:"account_source"`
	} `json:"Account"`
	AccessToken map[string]struct {
		HomeAccountID     string `json:"home_account_id"`
		Environment       string `json:"environment"`
		Realm             string `json:"realm"`
		CredentialType    string `json:"credential_type"`
		ClientID          string `json:"client_id"`
		Secret            string `json:"secret"`
		Target            string `json:"target"`
		ExpiresOn         string `json:"expires_on"`
		ExtendedExpiresOn string `json:"extended_expires_on"`
		CachedAt          string `json:"cached_at"`
		TokenType         string `json:"token_type"`
	} `json:"AccessToken"`
	RefreshToken map[string]struct {
		HomeAccountID  string `json:"home_account_id"`
		Environment    string `json:"environment"`
		CredentialType string `json:"credential_type"`
		ClientID       string `json:"client_id"`
		FamilyID       string `json:"family_id"`
		Secret         string `json:"secret"`
		Target         string `json:"target"`
	} `json:"RefreshToken"`
	IdToken map[string]struct {
		HomeAccountID  string `json:"home_account_id"`
		Environment    string `json:"environment"`
		CredentialType string `json:"credential_type"`
		ClientID       string `json:"client_id"`
		Secret         string `json:"secret"`
		Realm          string `json:"realm"`
	} `json:"IdToken"`
	AppMetadata map[string]struct {
		Environment string `json:"environment"`
		ClientID    string `json:"client_id"`
		FamilyID    string `json:"family_id"`
	} `json:"AppMetadata"`
}

func (l LocalCreds) First() interface{} {
	for _, c := range l.RefreshToken {
		return c
	}
	for _, c := range l.Account {
		return c
	}
	return nil
}

func (l LocalCreds) AssertionForUser(user string) string {
	for _, a := range l.Account {
		if strings.EqualFold(a.Username, user) {
			for _, t := range l.IdToken {
				if t.HomeAccountID == a.HomeAccountID {
					log.Debugln("id token cache hit")
					return t.Secret
				}
			}
			return ""
		}
	}
	return ""
}

// LoadLocalCreds parses the token cache, yielding the zero value when it is
// absent or unreadable. A malformed cache is a recoverable condition: the caller
// simply has no cached identities and can sign in again, so it must not end the
// process.
func LoadLocalCreds() (creds LocalCreds) {
	out, err := os.ReadFile(cachePath())
	if err != nil {
		log.Debugf("token cache unreadable: %v", err)
		return
	}
	if err = json.Unmarshal(out, &creds); err != nil {
		log.Debugf("token cache malformed, treating as empty: %v", err)
		return LocalCreds{}
	}
	return
}
