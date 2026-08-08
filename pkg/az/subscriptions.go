package az

import (
	"context"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

// newTenantsClient, newSubscriptionsClient and newGraphClient indirect the SDK
// client constructors so specs can substitute a credential-recording double
// without touching the enumeration logic itself.
var (
	newTenantsClient       = armsubscriptions.NewTenantsClient
	newSubscriptionsClient = armsubscriptions.NewClient
	newGraphClient         = msgraphsdk.NewGraphServiceClientWithCredentials
)

// Organization represents an Azure AD organization with detailed information
type Organization struct {
	ID                    string           `json:"id"`
	DisplayName           string           `json:"displayName"`
	VerifiedDomains       []VerifiedDomain `json:"verifiedDomains"`
	IsResourceTenant      bool             `json:"isResourceTenant"`
	TenantType            string           `json:"tenantType"`
	DefaultDomain         string           `json:"defaultDomain"`
	TenantBrandingLogoURL string           `json:"tenantBrandingLogoUrl,omitempty"`
}

type VerifiedDomain struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
	IsInitial bool   `json:"isInitial"`
}

// ListSubscriptionsCLI returns the profile's subscriptions, rebuilding it first
// when asked, when it is missing, or when it turned out to be empty. Errors are
// returned so a caller can react; ending the process here would strand the lock
// files and deferred cleanup the credential paths rely on.
//
// Any rebuild is scoped to hint, so a refresh requested under one identity can
// never re-enumerate as another one.
func ListSubscriptionsCLI(ctx context.Context, refresh bool, hint string) ([]cli.Subscription, error) {
	p, err := profilePath()
	if err != nil {
		return nil, err
	}
	if _, serr := os.Stat(p); errors.Is(serr, os.ErrNotExist) || refresh {
		if err = BuildProfile(ctx, hint); err != nil {
			return nil, err
		}
	}
	o, err := cli.LoadProfile(p)
	if err != nil {
		return nil, err
	}
	// Emptiness is judged after filtering: a profile full of another identity's
	// subscriptions is empty as far as this identity is concerned, and the
	// caller deserves a real enumeration rather than an empty list.
	if len(FilterSubscriptionsByUser(o.Subscriptions, hint)) == 0 {
		if err = BuildProfile(ctx, hint); err != nil {
			return nil, err
		}
		if o, err = cli.LoadProfile(p); err != nil {
			return nil, err
		}
	}
	return FilterSubscriptionsByUser(o.Subscriptions, hint), nil
}

// ListSubscriptions enumerates every subscription reachable from the Selected
// Account. It returns the list rather than writing the profile so the caller can
// merge it with what other identities previously contributed.
func ListSubscriptions(ctx context.Context, hint string) ([]cli.Subscription, error) {
	return NewEnumeratorForAccount(hint).ListSubscriptions(ctx)
}

// ListSubscriptions enumerates through the Enumerator so the tenant listing and
// each tenant's subscription listing happen once per invocation.
func (e *Enumerator) ListSubscriptions(ctx context.Context) (subscriptions []cli.Subscription, err error) {
	accounts, err := e.Accounts(ctx)
	if err != nil {
		return nil, err
	}
	// Attribution must name the identity that acquired the tokens, so it reads
	// the same hint the credentials were built from rather than repeating an
	// independent Active Account lookup.
	user := SubscriptionUser(ctx, accounts, e.hint)

	tenants, err := e.Tenants(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tenants {
		if t.TenantID == nil {
			continue
		}
		tenantSubs, terr := e.Subscriptions(ctx, *t.TenantID)
		if terr != nil {
			// One unreachable tenant must not sink the whole enumeration; the
			// user still needs the subscriptions the other tenants returned.
			log.Warnf("skipping tenant %s: %v", *t.TenantID, terr)
			continue
		}
		// Emulate --allow-no-subscriptions per tenant. The original check tested
		// the accumulated slice, so a tenant with no subscriptions was silently
		// dropped whenever an earlier tenant had contributed any.
		if len(tenantSubs) == 0 {
			subscriptions = append(subscriptions, cli.Subscription{
				EnvironmentName: "AzureCloud",
				ID:              *t.TenantID,
				Name:            "N/A(tenant level account)",
				State:           "Enabled",
				TenantID:        *t.TenantID,
				User:            &cli.User{Name: user, Type: "user"},
			})
			continue
		}
		for _, s := range tenantSubs {
			if s.SubscriptionID == nil {
				continue
			}
			name := ""
			if s.DisplayName != nil {
				name = *s.DisplayName
			}
			state := ""
			if s.State != nil {
				state = string(*s.State)
			}
			subscriptions = append(subscriptions, cli.Subscription{
				EnvironmentName: "AzureCloud",
				ID:              *s.SubscriptionID,
				Name:            name,
				State:           state,
				TenantID:        *t.TenantID,
				User:            &cli.User{Name: user, Type: "user"},
			})
		}
	}
	return
}

func ListSubscriptionsForTenant(ctx context.Context, tenant, hint string) (subscriptions []*armsubscriptions.Subscription, err error) {
	client, err := newSubscriptionsClient(TokenCredential{TenantID: tenant, PreferredUsername: hint}, nil)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(nil)
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to advance subscription page: %w", err)
		}

		// Filter out subscriptions Azure has retired. An absent State is not a
		// reason to drop the subscription; the field is optional, and the old
		// code silently discarded every entry that omitted it.
		for _, sub := range nextResult.Value {
			if sub.State != nil {
				switch *sub.State {
				case armsubscriptions.SubscriptionStateDisabled, armsubscriptions.SubscriptionStateDeleted:
					continue
				}
			}
			subscriptions = append(subscriptions, sub)
		}
	}
	return
}

func ListTenants(ctx context.Context, hint string) (tenants []*armsubscriptions.TenantIDDescription, err error) {
	client, err := newTenantsClient(TokenCredential{PreferredUsername: hint}, nil)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(nil)
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to advance tenant page: %w", err)
		}
		tenants = append(tenants, resp.Value...)
	}

	return
}

// ListOrganizations gets detailed information about all organizations (tenants)
// the user has access to, similar to what you see in "Switch Organizations"
func ListOrganizations(ctx context.Context, hint string) ([]Organization, error) {
	return NewEnumeratorForAccount(hint).ListOrganizations(ctx)
}

// ListOrganizations shares the Enumerator's tenant and subscription listings, so
// resolving "is this a resource tenant" costs no extra API calls.
func (e *Enumerator) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var organizations []Organization

	tenants, err := e.Tenants(ctx)
	if err != nil {
		return nil, err
	}

	for _, tenant := range tenants {
		if tenant.TenantID == nil {
			continue
		}

		// Get detailed organization info from Microsoft Graph
		org, err := e.organizationDetails(ctx, *tenant.TenantID)
		if err != nil {
			log.Warnf("Failed to get organization details for tenant %s: %v", *tenant.TenantID, err)
			// Add basic info if Graph call fails
			organizations = append(organizations, Organization{
				ID:          *tenant.TenantID,
				DisplayName: *tenant.TenantID, // Fallback to ID
				TenantType:  "Unknown",
			})
			continue
		}

		organizations = append(organizations, org)
	}

	return organizations, nil
}

// organizationDetails calls Microsoft Graph API to get detailed organization information
func (e *Enumerator) organizationDetails(ctx context.Context, tenantID string) (Organization, error) {
	// Create a tenant-specific credential, scoped to this Enumerator's identity.
	cred := TokenCredential{TenantID: tenantID, PreferredUsername: e.hint}

	// Create Microsoft Graph client
	client, err := newGraphClient(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return Organization{}, fmt.Errorf("failed to create Graph client for tenant %s: %w", tenantID, err)
	}

	// Get organization details
	organizations, err := client.Organization().Get(ctx, nil)
	if err != nil {
		return Organization{}, fmt.Errorf("failed to get organization details for tenant %s: %w", tenantID, err)
	}

	if organizations == nil || organizations.GetValue() == nil || len(organizations.GetValue()) == 0 {
		return Organization{}, fmt.Errorf("no organization found for tenant %s", tenantID)
	}

	// Convert from Graph SDK model to our model
	graphOrg := organizations.GetValue()[0]

	// Every Graph field here is optional. Fall back to the tenant id rather than
	// dereferencing a nil pointer and panicking mid-enumeration.
	org := Organization{
		ID:               tenantID,
		IsResourceTenant: e.hasSubscriptions(ctx, tenantID),
	}
	if id := graphOrg.GetId(); id != nil {
		org.ID = *id
	}
	if name := graphOrg.GetDisplayName(); name != nil {
		org.DisplayName = *name
	}

	// Extract verified domains
	if verifiedDomains := graphOrg.GetVerifiedDomains(); verifiedDomains != nil {
		for _, domain := range verifiedDomains {
			var vd VerifiedDomain
			if n := domain.GetName(); n != nil {
				vd.Name = *n
			}
			if d := domain.GetIsDefault(); d != nil {
				vd.IsDefault = *d
			}
			if i := domain.GetIsInitial(); i != nil {
				vd.IsInitial = *i
			}
			org.VerifiedDomains = append(org.VerifiedDomains, vd)

			// Set default domain
			if vd.IsDefault {
				org.DefaultDomain = vd.Name
			}
		}
	}

	// Set tenant type if available
	if tenantType := graphOrg.GetTenantType(); tenantType != nil {
		org.TenantType = *tenantType
	}

	// Set default domain from verified domains
	for _, domain := range org.VerifiedDomains {
		if domain.IsDefault {
			org.DefaultDomain = domain.Name
			break
		}
	}

	return org, nil
}

// hasSubscriptions reports whether the tenant holds any subscription, making it
// a "resource tenant". The listing is memoised, so this is free once the profile
// enumeration has already visited the tenant.
func (e *Enumerator) hasSubscriptions(ctx context.Context, tenantID string) bool {
	subscriptions, err := e.Subscriptions(ctx, tenantID)
	if err != nil {
		log.Debugf("tenant %s subscription probe failed: %v", tenantID, err)
		return false
	}
	return len(subscriptions) > 0
}

// ListTenantDetails returns basic tenant information with subscription counts
// This is a simpler alternative that doesn't require Microsoft Graph API permissions
func ListTenantDetails(ctx context.Context, hint string) ([]TenantDetail, error) {
	return NewEnumeratorForAccount(hint).ListTenantDetails(ctx)
}

// ListTenantDetails counts each tenant's subscriptions using the memoised
// listings, so the count costs one call per tenant rather than one per caller.
func (e *Enumerator) ListTenantDetails(ctx context.Context) ([]TenantDetail, error) {
	var details []TenantDetail

	tenants, err := e.Tenants(ctx)
	if err != nil {
		return nil, err
	}
	for _, tenant := range tenants {
		if tenant.TenantID == nil {
			continue
		}

		subscriptions, serr := e.Subscriptions(ctx, *tenant.TenantID)
		if serr != nil {
			log.Warnf("skipping tenant %s: %v", *tenant.TenantID, serr)
			continue
		}
		detail := TenantDetail{
			TenantID:          *tenant.TenantID,
			SubscriptionCount: len(subscriptions),
			HasSubscriptions:  len(subscriptions) > 0,
		}

		details = append(details, detail)
	}

	return details, nil
}

// TenantDetail represents basic tenant information
type TenantDetail struct {
	TenantID          string `json:"tenantId"`
	SubscriptionCount int    `json:"subscriptionCount"`
	HasSubscriptions  bool   `json:"hasSubscriptions"`
}
