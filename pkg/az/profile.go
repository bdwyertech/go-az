package az

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/google/uuid"
)

func BuildProfile() error {
	return BuildProfileWithUser("")
}

func BuildProfileWithUser(authenticatedUser string) error {
	f, err := cli.ProfilePath()
	if err != nil {
		return err
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
	currentSubs := p.Subscriptions
	newSubs := ListSubscriptionsWithUser(authenticatedUser)

	// Use a map to track unique subscriptions and prevent duplicates
	// Key: subscription ID, Value: subscription object
	subMap := make(map[string]cli.Subscription)

	// First, add all new subscriptions to the map
	for _, sub := range newSubs {
		fmt.Println("Found subscription:", sub.Name)
		subMap[sub.ID] = sub
	}

	// Then, add current subscriptions only if they don't already exist
	// This preserves subscriptions from other accounts while preventing duplicates
	for _, currentSub := range currentSubs {
		if _, exists := subMap[currentSub.ID]; !exists {
			subMap[currentSub.ID] = currentSub
		}
	}

	// Convert map back to slice
	p.Subscriptions = make([]cli.Subscription, 0, len(subMap))
	for _, sub := range subMap {
		p.Subscriptions = append(p.Subscriptions, sub)
	}

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

	return WriteProfile(p, f)
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
