//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.PersistentFlags().StringP("output", "o", "json", "Output format.  Allowed values: json  Default: json.")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	cobra.OnInitialize(initConfig)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:    "az",
	Hidden: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
	},
}

func initConfig() {
	// Environment Variable Munging
	viper.SetEnvPrefix("GOAZ")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if viper.GetBool("debug") || viper.GetBool("trace") {
		log.SetLevel(log.DebugLevel)
		if viper.GetBool("trace") {
			log.SetLevel(log.TraceLevel)
			log.SetReportCaller(true)
		}
	}
}
