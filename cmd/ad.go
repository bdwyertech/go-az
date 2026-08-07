//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
)

func init() {
	adCmd.AddCommand(adSignedInUserCmd)
	rootCmd.AddCommand(adCmd)
}

var adCmd = &cobra.Command{
	Use:   "ad",
	Short: "Manage Azure Active Directory Graph entities needed for Role Based Access Control.",
}

var adSignedInUserCmd = &cobra.Command{
	Use:       "signed-in-user",
	Short:     "Show graph information about current signed-in user in CLI.",
	ValidArgs: []string{"show"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "show":
			u, err := az.GetSignedInUser(cmd.Context(), "")
			if err != nil {
				return err
			}
			jsonBytes, err := json.MarshalIndent(u, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsonBytes))
			return nil
		default:
			return fmt.Errorf("unsupported argument: %s", args[0])
		}
	},
}
