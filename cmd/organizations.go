//
// Go AZ - Organizations
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
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.Flags().Bool("json", false, "Output in JSON format")
}

var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "List organizations you have access to",
	Long:  `List all Azure AD organizations (tenants) you have access to, similar to "Switch Organizations" in the Azure portal`,
	Run: func(cmd *cobra.Command, args []string) {
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
			fmt.Printf("%-36s %-40s %-30s %-15s %s\n", "Tenant ID", "Display Name", "Default Domain", "Has Resources", "Tenant Type")
			fmt.Println("----------------------------------------------------------------------------------------------------")

			for _, org := range organizations {
				hasResources := "No"
				if org.IsResourceTenant {
					hasResources = "Yes"
				}

				fmt.Printf("%-36s %-40s %-30s %-15s %s\n",
					org.ID,
					org.DisplayName,
					org.DefaultDomain,
					hasResources,
					org.TenantType)
			}
		}
	},
}
