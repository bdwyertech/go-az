package az

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
	"github.com/gofrs/flock"
	"github.com/mitchellh/go-homedir"
)

var credCache *Cache

func init() {
	cpath := cachePath()
	credCache = &Cache{
		path:  cpath,
		mutex: flock.New(cpath + ".lock"),
	}
}

func cacheDir() (d string) {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	d = filepath.Join(home, ".azure")
	if _, err = os.Stat(d); errors.Is(err, os.ErrNotExist) {
		os.Mkdir(d, 0750)
	}
	return
}

func cachePath() string {
	return filepath.Join(cacheDir(), "go_msal_token_cache.json")
}

type Cache struct {
	path   string
	mutex  *flock.Flock
	locked bool
	bytes  []byte
}

func (c *Cache) Export(ctx context.Context, m cache.Marshaler, k cache.ExportHints) error {
	jsonBytes, err := m.Marshal()
	if err != nil {
		log.Error(err)
		return nil
	}
	b := new(bytes.Buffer)
	json.Indent(b, jsonBytes, "", "  ")
	if bytes.Equal(c.bytes, b.Bytes()) {
		// log.Debug("cache: already up to date")
		return nil
	}
	// log.Debug("cache: acquiring write lock")
	if err = c.mutex.Lock(); err != nil {
		log.Error(err)
		return nil
	}
	// log.Debug("cache: write lock acquired")
	defer c.mutex.Unlock()
	if err = os.WriteFile(c.path, b.Bytes(), os.ModePerm); err != nil {
		log.Error(err)
	}
	return nil
}

func (c *Cache) Replace(ctx context.Context, u cache.Unmarshaler, k cache.ReplaceHints) error {
	// log.Debug("cache: acquiring read lock")
	if err := c.mutex.RLock(); err != nil {
		log.Error(err)
		return nil
	}
	// log.Debug("cache: read lock acquired")
	defer c.mutex.Unlock()
	var err error
	c.bytes, err = os.ReadFile(c.path)
	if err != nil {
		return nil
	}
	if err = u.Unmarshal(c.bytes); err != nil {
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
	} `json:"AccessToken"`
	RefreshToken map[string]struct {
		HomeAccountID  string `json:"home_account_id"`
		Environment    string `json:"environment"`
		CredentialType string `json:"credential_type"`
		ClientID       string `json:"client_id"`
		FamilyID       string `json:"family_id"`
		Secret         string `json:"secret"`
	} `json:"RefreshToken"`
	IdToken map[string]struct {
		HomeAccountID  string `json:"home_account_id"`
		Environment    string `json:"environment"`
		CredentialType string `json:"credential_type"`
		ClientID       string `json:"client_id"`
		Secret         string `json:"secret"`
	} `json:"IdToken"`
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
					log.Info("HIT!")
					return t.Secret
				}
			}
			return ""
		}
	}
	return ""
}

func LoadLocalCreds() (creds LocalCreds) {
	out, err := os.ReadFile(cachePath())
	if err != nil {
		// log.Debugln(err)
		return
	}
	if err = json.Unmarshal(out, &creds); err != nil {
		log.Fatal(err)
	}
	return
}

func UserForTenant(tenant string) string {
	var common string
	for _, a := range LoadLocalCreds().Account {
		switch a.Realm {
		case tenant:
			return a.Username
		case "common", "organizations":
			common = a.Username
		}
	}
	return common
}
