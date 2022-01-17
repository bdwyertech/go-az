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

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	accountGetAccessTokenCmd.Flags().StringP("name", "n", "", "Name of subscription.")
	accountGetAccessTokenCmd.Flags().StringP("subscription", "s", "", "ID of subscription.")
	accountGetAccessTokenCmd.Flags().StringP("resource", "", "", "Azure resource endpoints in AAD v1.0.")
	accountGetAccessTokenCmd.Flags().StringP("resource-type", "", "", "Type of well-known resource.  Allowed values: aad-graph, arm, batch, data-lake, media, ms-graph, oss-rdbms.")
	accountGetAccessTokenCmd.Flags().StringSliceP("scope", "", []string{}, "Space-separated AAD scopes in AAD v2.0. Default to Azure Resource Manager.")
	accountGetAccessTokenCmd.Flags().StringP("tenant", "t", "", "Tenant ID for which the token is acquired. Only available for user and service principal account, not for MSI or Cloud Shell account.")

	accountShowCmd.Flags().StringP("name", "n", "", "Name of subscription.")
	accountShowCmd.Flags().StringP("subscription", "s", "", "ID of subscription.")

	accountCmd.AddCommand(
		accountCachedCmd,
		accountGetAccessTokenCmd,
		accountListCmd,
		accountShowCmd,
	)
	rootCmd.AddCommand(accountCmd)
}

var accountCmd = &cobra.Command{
	Use: "account",
}

var accountShowCmd = &cobra.Command{
	Use: "show",
	// List Current Subscription
	Run: func(cmd *cobra.Command, args []string) {
		// o := az.ListSubscriptions()
		var defaultSub interface{}
		for _, s := range az.ListSubscriptionsCLI() {
			if s.IsDefault {
				defaultSub = s
				break
			}
		}
		jsonBytes, err := json.MarshalIndent(defaultSub, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}

var accountListCmd = &cobra.Command{
	Use: "list",
	// List All Subscriptions
	Run: func(cmd *cobra.Command, args []string) {
		// o := az.ListSubscriptions()
		o := az.ListSubscriptionsCLI()
		jsonBytes, err := json.MarshalIndent(o, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}

var accountGetAccessTokenCmd = &cobra.Command{
	Use: "get-access-token",
	Run: func(cmd *cobra.Command, args []string) {
		viper.BindPFlags(cmd.Flags())
		u, err := az.GetAccessToken(cmd.Context(), az.AccessTokenOptions{
			Resource:       viper.GetString("resource"),
			SubscriptionID: viper.GetString("subscription"),
			Tenant:         viper.GetString("tenant"),
		})
		if err != nil {
			log.Fatal(err)
		}
		jsonBytes, err := json.MarshalIndent(u, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}

var accountCachedCmd = &cobra.Command{
	Use: "cached",
	Run: func(cmd *cobra.Command, args []string) {
		cached := az.GetCachedAccounts()
		jsonBytes, err := json.MarshalIndent(cached, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}

// az account get-access-token

// az account show
//
// az account show -s subscription-id
//
// az account list
