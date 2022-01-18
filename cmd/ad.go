//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"az/pkg/az"
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	adCmd.AddCommand(
		adSignedInUserCmd,
	)
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
	Args:      cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "show":
			u := az.GetSignedInUser("")
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
