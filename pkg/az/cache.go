package az

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

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
	RefreshToken map[string]struct {
		HomeAccountID  string `json:"home_account_id"`
		Environment    string `json:"environment"`
		CredentialType string `json:"credential_type"`
		ClientID       string `json:"client_id"`
		FamilyID       string `json:"family_id"`
		Secret         string `json:"secret"`
	} `json:"RefreshToken"`
}

func (l LocalCreds) First() (id string) {
	for _, c := range l.RefreshToken {
		return c.HomeAccountID
	}
	for _, c := range l.Account {
		return c.HomeAccountID
	}
	return
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
