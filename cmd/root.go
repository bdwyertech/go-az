//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bdwyertech/go-az/pkg/az"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.PersistentFlags().StringP("output", "o", "json", "Output format.  Allowed values: json  Default: json.")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
	// No shorthand: -u is reserved for the Azure CLI's own --username.
	rootCmd.PersistentFlags().String("preferred-username", "",
		"Select a cached account by username, object ID, or home account ID. Falls back to GO_AZ_USERNAME then AZURE_USERNAME.")
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	cobra.OnInitialize(initConfig)
}

// accountHint resolves the Account Hint for the command being run. Cobra owns
// the flag, and az.ResolveAccountHint applies the environment fallbacks, so a
// sibling command that never set the flag resolves to whatever the environment
// says and nothing more.
func accountHint(cmd *cobra.Command) string {
	flag, _ := cmd.Flags().GetString("preferred-username")
	return az.ResolveAccountHint(flag)
}

// Execute runs the root command under a context that is cancelled on SIGINT or
// SIGTERM. Cancellation, rather than an immediate exit, is what lets an
// in-flight command unwind: deferred file locks are released and no partially
// written credential file is left behind. A second signal restores the default
// handler so an unresponsive command can still be killed.
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := rootCmd.ExecuteContext(ctx)
	stop()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:    "az",
	Hidden: true,
	// A command that fails at runtime has already parsed its arguments
	// correctly, so dumping the usage text only buries the actual error and
	// pollutes the output a caller may be parsing.
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		_ = viper.BindPFlags(cmd.Flags())
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
