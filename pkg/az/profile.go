package az

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Azure/go-autorest/autorest/azure/cli"
	"github.com/google/uuid"
)

// profilePath resolves the Azure CLI profile. Tests redirect the credential
// directory, so honour the override before falling back to the CLI's own
// resolution (which respects AZURE_CONFIG_DIR).
func profilePath() (string, error) {
	if credDirOverride != "" {
		d, err := cacheDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(d, "azureProfile.json"), nil
	}
	return cli.ProfilePath()
}

// BuildProfile refreshes the Azure CLI profile. The whole read-modify-write
// runs under one exclusive advisory lock so a concurrent go-az or Azure CLI
// cannot interleave and lose the default-subscription selection.
func BuildProfile(ctx context.Context) error {
	f, err := profilePath()
	if err != nil {
		return err
	}
	return withExclusiveLock(ctx, f+".lock", func() error {
		p, _ := cli.LoadProfile(f)
		if p.InstallationID == "" {
			p.InstallationID = uuid.NewString()
		}
		// Enumeration only sees what the Selected Account can reach, so union it
		// with the existing profile. MergeSubscriptions also settles ordering and
		// the single surviving default.
		discovered, err := ListSubscriptions(ctx)
		if err != nil {
			return err
		}
		p.Subscriptions = MergeSubscriptions(p.Subscriptions, discovered)

		return WriteProfile(p, f)
	})
}

func WriteProfile(profile cli.Profile, path string) (err error) {
	profileBytes, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding Profile %s: %w", path, err)
	}
	if err = WriteFileAtomic(path, profileBytes, credFileMode); err != nil {
		return fmt.Errorf("error writing Profile %s: %w", path, err)
	}
	return
}

func LoadProfile() (p cli.Profile, err error) {
	f, err := profilePath()
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
