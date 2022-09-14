package az

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/google/uuid"
)

func BuildProfile() {
	f, err := cli.ProfilePath()
	if err != nil {
		log.Fatal(err)
	}
	var p cli.Profile
	// Try to get the default
	var defaultSub string
	p, _ = cli.LoadProfile(f)
	for _, s := range p.Subscriptions {
		if s.IsDefault {
			defaultSub = s.ID
			break
		}
	}
	if p.InstallationID == "" {
		p.InstallationID = uuid.NewString()
	}
	p.Subscriptions = ListSubscriptions()

	if defaultSub != "" {
		for i, s := range p.Subscriptions {
			if s.ID == defaultSub {
				p.Subscriptions[i].IsDefault = true
				break
			}
		}
	} else if len(p.Subscriptions) > 0 {
		p.Subscriptions[0].IsDefault = true
	}

	WriteProfile(p, f)
}

func WriteProfile(profile cli.Profile, path string) (err error) {
	profileBytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding Profile %s: %w", path, err)
	}
	if err = os.WriteFile(path, profileBytes, os.ModePerm); err != nil {
		return fmt.Errorf("error writing Profile %s: %w", path, err)
	}
	return
}

func LoadProfile() (p cli.Profile, err error) {
	f, err := cli.ProfilePath()
	if err != nil {
		return
	}
	return cli.LoadProfile(f)
}

func DefaultSubscription() (id string) {
	p, err := LoadProfile()
	if err != nil {
		return
	}
	for _, p := range p.Subscriptions {
		if p.IsDefault {
			return p.ID
		}
	}
	return
}
