package az

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

var credCache Cache

func cachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(home, ".azure", "go_msal_token_cache.json")
}

type Cache struct{}

func (c Cache) Export(m cache.Marshaler, k string) {
	if k != "" {
		log.Fatal(k)
	}
	cachePath := cachePath()
	jsonBytes, err := m.Marshal()
	if err != nil {
		log.Error(err)
		return
	}
	b := new(bytes.Buffer)
	json.Indent(b, jsonBytes, "", "  ")
	if err = os.WriteFile(cachePath, b.Bytes(), os.ModePerm); err != nil {
		log.Error(err)
	}
}

func (c Cache) Replace(u cache.Unmarshaler, k string) {
	if k != "" {
		log.Fatal(k)
	}
	out, err := os.ReadFile(cachePath())
	if err != nil {
		return
	}
	if err = u.Unmarshal(out); err != nil {
		log.Error(err)
	}
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
