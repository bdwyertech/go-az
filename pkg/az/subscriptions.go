package az

import (
	"context"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/Azure/go-autorest/autorest/azure/cli"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
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

func ListSubscriptionsCLI(refresh bool) []cli.Subscription {
	p, err := cli.ProfilePath()
	if err != nil {
		log.Fatal(err)
	}
	if _, err = os.Stat(p); errors.Is(err, os.ErrNotExist) || refresh {
		if err = BuildProfile(); err != nil {
			log.Fatal(err)
		}
	}
	o, err := cli.LoadProfile(p)
	if err != nil {
		log.Fatal(err)
	}
	if len(o.Subscriptions) == 0 {
		if err = BuildProfile(); err != nil {
			log.Fatal(err)
		}
	}
	return o.Subscriptions
}

func ListSubscriptions() (subscriptions []cli.Subscription) {
	for _, t := range ListTenants() {
		for _, s := range ListSubscriptionsForTenant(*t.TenantID) {
			subscriptions = append(subscriptions, cli.Subscription{
				EnvironmentName: "AzureCloud",
				ID:              *s.SubscriptionID,
				IsDefault:       false,
				Name:            *s.DisplayName,
				State:           string(*s.State),
				TenantID:        *t.TenantID,
				User: &cli.User{
					Name: UserForTenant(*t.TenantID),
					Type: "user",
				},
			})
		}
	}
	return
}

func ListSubscriptionsForTenant(tenant string) (subscriptions []*armsubscription.Subscription) {
	client, err := armsubscription.NewSubscriptionsClient(TokenCredential{TenantID: tenant}, nil)
	if err != nil {
		log.Fatal(err)
	}
	pager := client.NewListPager(nil)
	for pager.More() {
		nextResult, err := pager.NextPage(context.Background())
		if err != nil {
			log.Fatalln("failed to advance page:", err)
		}
		subscriptions = append(subscriptions, nextResult.Value...)
	}
	// TODO: Ensure we only return "enabled" subscriptions
	return
}

func ListTenants() (tenants []*armsubscription.TenantIDDescription) {
	client, err := armsubscription.NewTenantsClient(TokenCredential{}, nil)
	if err != nil {
		log.Fatal(err)
	}
	pager := client.NewListPager(nil)
	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			log.Fatalln("failed to advance page:", err)
		}
		tenants = append(tenants, resp.Value...)
	}

	return
}

// ListOrganizations gets detailed information about all organizations (tenants)
// the user has access to, similar to what you see in "Switch Organizations"
func ListOrganizations() ([]Organization, error) {
	var organizations []Organization

	// Get all tenants the user has access to
	tenants := ListTenants()

	for _, tenant := range tenants {
		if tenant.TenantID == nil {
			continue
		}

		// Get detailed organization info from Microsoft Graph
		org, err := getOrganizationDetails(*tenant.TenantID)
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

// getOrganizationDetails calls Microsoft Graph API to get detailed organization information
func getOrganizationDetails(tenantID string) (Organization, error) {
	// Create a tenant-specific credential
	cred := TokenCredential{TenantID: tenantID}

	// Create Microsoft Graph client
	client, err := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	if err != nil {
		return Organization{}, fmt.Errorf("failed to create Graph client for tenant %s: %w", tenantID, err)
	}

	// Get organization details
	organizations, err := client.Organization().Get(context.Background(), nil)
	if err != nil {
		return Organization{}, fmt.Errorf("failed to get organization details for tenant %s: %w", tenantID, err)
	}

	if organizations == nil || organizations.GetValue() == nil || len(organizations.GetValue()) == 0 {
		return Organization{}, fmt.Errorf("no organization found for tenant %s", tenantID)
	}

	// Convert from Graph SDK model to our model
	graphOrg := organizations.GetValue()[0]

	org := Organization{
		ID:               *graphOrg.GetId(),
		DisplayName:      *graphOrg.GetDisplayName(),
		IsResourceTenant: hasSubscriptions(tenantID),
	}

	// Extract verified domains
	if verifiedDomains := graphOrg.GetVerifiedDomains(); verifiedDomains != nil {
		for _, domain := range verifiedDomains {
			vd := VerifiedDomain{
				Name:      *domain.GetName(),
				IsDefault: *domain.GetIsDefault(),
				IsInitial: *domain.GetIsInitial(),
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

// hasSubscriptions checks if the tenant has any subscriptions (making it a "resource tenant")
func hasSubscriptions(tenantID string) bool {
	subscriptions := ListSubscriptionsForTenant(tenantID)
	return len(subscriptions) > 0
}

// ListTenantDetails returns basic tenant information with subscription counts
// This is a simpler alternative that doesn't require Microsoft Graph API permissions
func ListTenantDetails() []TenantDetail {
	var details []TenantDetail

	tenants := ListTenants()
	for _, tenant := range tenants {
		if tenant.TenantID == nil {
			continue
		}

		subscriptions := ListSubscriptionsForTenant(*tenant.TenantID)
		detail := TenantDetail{
			TenantID:          *tenant.TenantID,
			SubscriptionCount: len(subscriptions),
			HasSubscriptions:  len(subscriptions) > 0,
		}

		details = append(details, detail)
	}

	return details
}

// TenantDetail represents basic tenant information
type TenantDetail struct {
	TenantID          string `json:"tenantId"`
	SubscriptionCount int    `json:"subscriptionCount"`
	HasSubscriptions  bool   `json:"hasSubscriptions"`
}
