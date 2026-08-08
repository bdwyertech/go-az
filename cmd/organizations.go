//
// Go AZ - Organizations
//
// Copyright © 2025 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// resolveIdentity and listOrganizations are indirected so a spec can drive the
// command end to end against a fake cache instead of a live tenant.
var (
	resolveIdentity = func(cmd *cobra.Command, hint string) (string, error) {
		return az.ResolveEnumerationIdentity(cmd.Context(), az.NewEnumerator(), hint)
	}
	listOrganizations = func(cmd *cobra.Command, username string) ([]az.Organization, error) {
		return az.ListOrganizations(cmd.Context(), username)
	}
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
		// Resolve the identity before the first API call so an unknown hint
		// aborts while stdout is still empty.
		username, err := resolveIdentity(cmd, accountHint(cmd))
		if err != nil {
			return fmt.Errorf("selecting an account: %w", err)
		}
		emitAttribution(cmd, username)

		organizations, err := listOrganizations(cmd, username)
		if err != nil {
			return fmt.Errorf("listing organizations: %w", err)
		}

		// Write through the command's stream, which is os.Stdout in production
		// and a buffer under test, so the payload stays separable from the
		// attribution line on stderr.
		if jsonOutput := viper.GetBool("json"); jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			if err := enc.Encode(organizations); err != nil {
				return fmt.Errorf("encoding organizations: %w", err)
			}
		} else {
			table := tablewriter.NewWriter(cmd.OutOrStdout())
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
