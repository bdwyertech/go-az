package az

import (
	"context"
	"fmt"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// ReportedAccount is one cached identity as reported to the user.
type ReportedAccount struct {
	Username      string `json:"username"`
	HomeAccountID string `json:"homeAccountId"`
	Tenant        string `json:"tenant"`
	IsActive      bool   `json:"isActive"`
}

// AccountReport summarizes who is signed in locally.
type AccountReport struct {
	ActiveUsername      string            `json:"activeUsername"`
	ActiveHomeAccountID string            `json:"activeHomeAccountId"`
	ActiveIsCached      bool              `json:"activeIsCached"`
	Accounts            []ReportedAccount `json:"accounts"`
}

// BuildAccountReport pairs a cache snapshot with the recorded Active Account.
func BuildAccountReport(ctx context.Context, accounts []public.Account) (AccountReport, error) {
	s, err := LoadState(ctx)
	if err != nil {
		return AccountReport{}, err
	}

	r := AccountReport{
		ActiveUsername:      s.ActiveUsername,
		ActiveHomeAccountID: s.ActiveHomeAccountID,
		Accounts:            make([]ReportedAccount, 0, len(accounts)),
	}
	for _, a := range accounts {
		active := s.ActiveHomeAccountID != "" && a.HomeAccountID == s.ActiveHomeAccountID
		r.ActiveIsCached = r.ActiveIsCached || active
		r.Accounts = append(r.Accounts, ReportedAccount{
			Username:      a.PreferredUsername,
			HomeAccountID: a.HomeAccountID,
			Tenant:        homeTenant(a),
			IsActive:      active,
		})
	}
	return r, nil
}

// SetActiveAccount records the account named by hint as the Active Account. No
// token is acquired, so the hint must already name a cached account: a typo
// must not leave the state file pointing at an identity that does not exist.
func SetActiveAccount(ctx context.Context, accounts []public.Account, hint string) (public.Account, error) {
	if hint == "" {
		return public.Account{}, fmt.Errorf("%w: no user supplied", ErrNoMatchingAccount)
	}

	selected, err := ResolveAccount(accounts, hint, "", "")
	if err != nil {
		return public.Account{}, err
	}

	if err := StoreState(ctx, State{
		ActiveUsername:      selected.PreferredUsername,
		ActiveHomeAccountID: selected.HomeAccountID,
	}); err != nil {
		return public.Account{}, err
	}
	return selected, nil
}

// RecordActiveAccount stores the identity a completed login authenticated. An
// account with no home account id is ignored rather than recorded, because a
// blank pointer would silently disable Active Account resolution.
func RecordActiveAccount(ctx context.Context, a public.Account) error {
	if a.HomeAccountID == "" {
		return nil
	}
	return StoreState(ctx, State{
		ActiveUsername:      a.PreferredUsername,
		ActiveHomeAccountID: a.HomeAccountID,
	})
}
