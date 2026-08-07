//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var ReleaseVer, ReleaseDate, GitCommit string

var versionCmd = &cobra.Command{
	Use: "version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("TF_PLUGIN_MAGIC_COOKIE") != "" {
			// Terraform expects this output to look like actual Azure CLI JSON output
			fmt.Println(`{"azure-cli": "2.32.0"}`)
			return nil
		}
		ver, err := json.MarshalIndent(struct {
			Version, Date, Commit, Runtime string
		}{
			ReleaseVer, ReleaseDate, GitCommit, runtime.Version(),
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(ver))
		return nil
	},
}
