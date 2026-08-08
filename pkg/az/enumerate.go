package az

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// Enumerator bounds the API and disk traffic of a single invocation.
//
// Discovery is naturally a fan-out: organizations, tenant details, and
// subscription listing all need the same tenant list, and deciding whether a
// tenant is a resource tenant needs the same subscription list that building the
// profile needs. Calling through this type memoises each distinct listing so a
// tenant is enumerated once, each tenant's subscriptions once, and the token
// cache parsed once, no matter how many callers ask.
//
// Errors are memoised alongside results so a failure is not retried in a loop.
type Enumerator struct {
	// hint is the Account Hint this Enumerator is scoped to for its whole
	// life; every credential it constructs carries this identity.
	hint string

	listTenants  func(context.Context) ([]*armsubscriptions.TenantIDDescription, error)
	listSubs     func(context.Context, string) ([]*armsubscriptions.Subscription, error)
	loadAccounts func(context.Context) ([]public.Account, error)

	// Call counters exist so specs can assert the bound rather than infer it.
	tenantCalls  int
	subCalls     map[string]int
	accountCalls int

	tenantsDone bool
	tenants     []*armsubscriptions.TenantIDDescription
	tenantsErr  error

	subs    map[string][]*armsubscriptions.Subscription
	subErrs map[string]error

	accountsDone bool
	accounts     []public.Account
	accountsErr  error
}

// NewEnumerator returns an Enumerator backed by the live Azure and cache
// calls, unscoped to any identity.
func NewEnumerator() *Enumerator {
	return NewEnumeratorForAccount("")
}

// NewEnumeratorForAccount returns an Enumerator backed by the live Azure and
// cache calls, scoped to hint for every credential it constructs.
func NewEnumeratorForAccount(hint string) *Enumerator {
	return &Enumerator{
		hint: hint,
		listTenants: func(ctx context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
			return ListTenants(ctx, hint)
		},
		listSubs: func(ctx context.Context, tenant string) ([]*armsubscriptions.Subscription, error) {
			return ListSubscriptionsForTenant(ctx, tenant, hint)
		},
		loadAccounts: GetCachedAccounts,
		subCalls:     map[string]int{},
		subs:         map[string][]*armsubscriptions.Subscription{},
		subErrs:      map[string]error{},
	}
}

// Tenants lists every tenant, at most once per Enumerator.
func (e *Enumerator) Tenants(ctx context.Context) ([]*armsubscriptions.TenantIDDescription, error) {
	if !e.tenantsDone {
		e.tenantCalls++
		e.tenants, e.tenantsErr = e.listTenants(ctx)
		e.tenantsDone = true
	}
	return e.tenants, e.tenantsErr
}

// Subscriptions lists one tenant's subscriptions, at most once per tenant.
func (e *Enumerator) Subscriptions(ctx context.Context, tenant string) ([]*armsubscriptions.Subscription, error) {
	if _, done := e.subs[tenant]; !done {
		if _, failed := e.subErrs[tenant]; !failed {
			e.subCalls[tenant]++
			subs, err := e.listSubs(ctx, tenant)
			if err != nil {
				e.subErrs[tenant] = err
				return nil, err
			}
			e.subs[tenant] = subs
		}
	}
	return e.subs[tenant], e.subErrs[tenant]
}

// Accounts returns the token cache snapshot, parsed at most once.
func (e *Enumerator) Accounts(ctx context.Context) ([]public.Account, error) {
	if !e.accountsDone {
		e.accountCalls++
		e.accounts, e.accountsErr = e.loadAccounts(ctx)
		e.accountsDone = true
	}
	return e.accounts, e.accountsErr
}
