package az

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// State is the go-az owned companion to azureProfile.json. It records which
// Account the user last selected so later invocations without an explicit hint
// stay on that identity. It is deliberately separate from the Azure CLI profile
// so go-az never writes fields the real CLI does not understand.
type State struct {
	ActiveUsername      string `json:"activeUsername"`
	ActiveHomeAccountID string `json:"activeHomeAccountId"`
}

// stateFileName is the State File's name within the credential directory.
const stateFileName = "go_az_state.json"

// statePath resolves the State File beside the Token Cache, honouring the
// credential directory override so specs stay hermetic.
func statePath() (string, error) {
	d, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, stateFileName), nil
}

// LoadState reads the State File under a shared lock. A missing or unparsable
// file yields the zero State without an error so a corrupt pointer can never
// wedge the tool; only unexpected I/O failures are reported.
func LoadState(ctx context.Context) (s State, err error) {
	p, err := statePath()
	if err != nil {
		return State{}, err
	}

	err = withSharedLock(ctx, p+".lock", func() error {
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if err := json.Unmarshal(b, &s); err != nil {
			log.Debugf("discarding unparsable state file %s: %v", p, err)
			s = State{}
		}
		return nil
	})
	if err != nil {
		return State{}, err
	}
	return s, nil
}

// StoreState writes the State File under an exclusive lock, replacing it
// atomically so a concurrent reader never observes a partial document.
func StoreState(ctx context.Context, s State) error {
	p, err := statePath()
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return withExclusiveLock(ctx, p+".lock", func() error {
		return WriteFileAtomic(p, b, credFileMode)
	})
}
