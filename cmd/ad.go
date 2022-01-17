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
	Use: "ad",
}

var adSignedInUserCmd = &cobra.Command{
	Use:       "signed-in-user",
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
		}
	},
}
