//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// {
// 	"azure-cli": "2.32.0",
// 	"azure-cli-core": "2.32.0",
// 	"azure-cli-telemetry": "1.0.6",
// 	"extensions": {}
// }

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`{
			"azure-cli": "2.32.0",
			"azure-cli-core": "2.32.0",
			"azure-cli-telemetry": "1.0.6",
			"extensions": {}
		  }`)
	},
}
