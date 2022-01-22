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

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var ReleaseVer, ReleaseDate, GitCommit string

var versionCmd = &cobra.Command{
	Use: "version",
	Run: func(cmd *cobra.Command, args []string) {
		if os.Getenv("TF_PLUGIN_MAGIC_COOKIE") != "" {
			// Terraform expects this output to look like actual Azure CLI JSON output
			fmt.Println(`{"azure-cli": "2.32.0"}`)
			return
		}
		ver, err := json.MarshalIndent(struct {
			Version, Date, Commit, Runtime string
		}{
			ReleaseVer, ReleaseDate, GitCommit, runtime.Version(),
		}, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(ver))
	},
}
