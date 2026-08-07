//
// Go AZ - Organizations
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
	rootCmd.AddCommand(organizationsCmd)
	organizationsCmd.Flags().Bool("json", false, "Output in JSON format")
}

var organizationsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "List organizations you have access to",
	Long:  `List all Azure AD organizations (tenants) you have access to, similar to "Switch Organizations" in the Azure portal`,
	RunE: func(cmd *cobra.Command, args []string) error {
		organizations, err := az.ListOrganizations(cmd.Context())
		if err != nil {
			return fmt.Errorf("listing organizations: %w", err)
		}

		if jsonOutput := viper.GetBool("json"); jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(organizations); err != nil {
				return fmt.Errorf("encoding organizations: %w", err)
			}
		} else {
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"Tenant ID", "Display Name", "Default Domain", "Has Subscriptions", "Tenant Type"})
			table.Configure(func(config *tablewriter.Config) {
				config.Row.Alignment.Global = tw.AlignLeft
			})

			for _, org := range organizations {
				hasSubscriptions := "No"
				if org.IsResourceTenant {
					hasSubscriptions = "Yes"
				}

				_ = table.Append([]string{
					org.ID,
					org.DisplayName,
					org.DefaultDomain,
					hasSubscriptions,
					org.TenantType,
				})
			}
			_ = table.Render()
		}
		return nil
	},
}
