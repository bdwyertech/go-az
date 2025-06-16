//
// Go AZ - Tenants
//
// Copyright © 2025 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(tenantsCmd)
	tenantsCmd.Flags().Bool("json", false, "Output in JSON format")
	tenantsCmd.Flags().Bool("detailed", false, "Get detailed organization information using Microsoft Graph API")
}

var tenantsCmd = &cobra.Command{
	Use:   "tenants",
	Short: "List tenant details",
	Long:  `List all Azure AD tenants you have access to with subscription counts`,
	Run: func(cmd *cobra.Command, args []string) {
		details := viper.GetBool("detailed")

		if details {
			// Use the detailed Graph API version
			organizations, err := az.ListOrganizations()
			if err != nil {
				fmt.Printf("Error listing organizations: %v\n", err)
				return
			}

			if jsonOutput := viper.GetBool("json"); jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(organizations); err != nil {
					fmt.Printf("Error marshaling JSON: %v\n", err)
					return
				}
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.Header([]string{"Tenant ID", "Display Name", "Default Domain", "Tenant Type"})
				table.Configure(func(config *tablewriter.Config) {
					config.Row.Alignment.Global = tw.AlignLeft
				})

				for _, org := range organizations {
					table.Append([]string{
						org.ID,
						org.DisplayName,
						org.DefaultDomain,
						org.TenantType,
					})
				}
				table.Render()
			}
		} else {
			// Use the simple version
			tenantDetails := az.ListTenantDetails()

			if jsonOutput := viper.GetBool("json"); jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(tenantDetails); err != nil {
					fmt.Printf("Error marshaling JSON: %v\n", err)
					return
				}
			} else {
				table := tablewriter.NewWriter(os.Stdout)
				table.Header([]string{"Tenant ID", "Subscriptions", "Has Subscriptions"})
				table.Configure(func(config *tablewriter.Config) {
					config.Row.Alignment.Global = tw.AlignLeft
				})

				for _, detail := range tenantDetails {
					hasResources := "No"
					if detail.HasSubscriptions {
						hasResources = "Yes"
					}

					table.Append([]string{
						detail.TenantID,
						fmt.Sprintf("%d", detail.SubscriptionCount),
						hasResources,
					})
				}
				table.Render()
			}
		}
	},
}
