//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

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
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "show":
			u := az.GetSignedInUser(cmd.Context(), "")
			jsonBytes, err := json.MarshalIndent(u, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonBytes))
		default:
			log.Fatalln("Unsupported argument:", args[0])
		}
	},
}
