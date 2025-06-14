//
// Go AZ - Tenants
//
// Copyright © 2025 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
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
		details, _ := cmd.Flags().GetBool("detailed")

		if details {
			// Use the detailed Graph API version
			organizations, err := az.ListOrganizations()
			if err != nil {
				fmt.Printf("Error listing organizations: %v\n", err)
				return
			}

			if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
				data, err := json.MarshalIndent(organizations, "", "  ")
				if err != nil {
					fmt.Printf("Error marshaling JSON: %v\n", err)
					return
				}
				fmt.Println(string(data))
			} else {
				fmt.Printf("%-36s %-40s %-30s %s\n", "Tenant ID", "Display Name", "Default Domain", "Tenant Type")
				fmt.Println("-----------------------------------------------------------------------------------------")

				for _, org := range organizations {
					fmt.Printf("%-36s %-40s %-30s %s\n",
						org.ID,
						org.DisplayName,
						org.DefaultDomain,
						org.TenantType)
				}
			}
		} else {
			// Use the simple version
			tenantDetails := az.ListTenantDetails()

			if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
				data, err := json.MarshalIndent(tenantDetails, "", "  ")
				if err != nil {
					fmt.Printf("Error marshaling JSON: %v\n", err)
					return
				}
				fmt.Println(string(data))
			} else {
				fmt.Printf("%-36s %-15s %s\n", "Tenant ID", "Subscriptions", "Has Resources")
				fmt.Println("----------------------------------------------------------------")

				for _, detail := range tenantDetails {
					hasResources := "No"
					if detail.HasSubscriptions {
						hasResources = "Yes"
					}

					fmt.Printf("%-36s %-15d %s\n",
						detail.TenantID,
						detail.SubscriptionCount,
						hasResources)
				}
			}
		}
	},
}
