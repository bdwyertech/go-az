//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	accountGetAccessTokenCmd.Flags().StringP("name", "n", "", "Name of subscription.")
	accountGetAccessTokenCmd.Flags().StringP("subscription", "s", "", "ID of subscription.")
	accountGetAccessTokenCmd.Flags().StringP("resource", "", "", "Azure resource endpoints in AAD v1.0.")
	accountGetAccessTokenCmd.Flags().StringSliceP("scope", "", []string{}, "Space-separated AAD scopes in AAD v2.0. Default to Azure Resource Manager.")
	accountGetAccessTokenCmd.Flags().StringP("tenant", "t", "", "Tenant ID for which the token is acquired. Only available for user and service principal account, not for MSI or Cloud Shell account.")
	accountGetAccessTokenCmd.Flags().StringP("client", "c", "", "Client Application ID for which the token is acquired.")

	accountShowCmd.Flags().StringP("name", "n", "", "Name of subscription.")
	accountShowCmd.Flags().StringP("subscription", "s", "", "ID of subscription.")

	accountListCmd.Flags().BoolP("refresh", "", false, "Refresh list of available subscriptions")

	accountCmd.AddCommand(
		accountCachedCmd,
		accountGetAccessTokenCmd,
		accountListCmd,
		accountShowCmd,
	)
	rootCmd.AddCommand(accountCmd)
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage Azure subscription information.",
}

var accountShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Get the details of a subscription.",
	// List Current Subscription
	Run: func(cmd *cobra.Command, args []string) {
		var sub interface{}
		subName, subId := viper.GetString("name"), viper.GetString("subscription")
		if subName != "" || subId != "" {
			for _, s := range az.ListSubscriptionsCLI(false) {
				if subId != "" && strings.EqualFold(subId, s.ID) {
					sub = s
					break
				}
				if subName != "" && strings.EqualFold(subName, s.Name) {
					sub = s
					break
				}
			}
			if sub == nil {
				log.Fatal("Unable to find matching subscription!")
			}
		} else {
			for _, s := range az.ListSubscriptionsCLI(false) {
				if s.IsDefault {
					sub = s
					break
				}
			}
		}

		jsonBytes, err := json.MarshalIndent(sub, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "Get a list of subscriptions for the logged in account.",
	// List All Subscriptions
	Run: func(cmd *cobra.Command, args []string) {
		s := az.ListSubscriptionsCLI(viper.GetBool("refresh"))
		jsonBytes, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}

var accountGetAccessTokenCmd = &cobra.Command{
	Use:   "get-access-token",
	Short: "Get a token for utilities to access Azure.",
	Run: func(cmd *cobra.Command, args []string) {
		u, err := az.GetAccessToken(cmd.Context(), az.AccessTokenOptions{
			Resource:       viper.GetString("resource"),
			Scope:          viper.GetStringSlice("scope"),
			SubscriptionID: viper.GetString("subscription"),
			Tenant:         viper.GetString("tenant"),
			Client:         viper.GetString("client"),
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
	Use:   "cached",
	Short: "List cached accounts.",
	Run: func(cmd *cobra.Command, args []string) {
		cached, err := az.GetCachedAccounts(cmd.Context())
		if err != nil {
			log.Fatal(err)
		}
		if len(cached) == 0 {
			fmt.Println("[]")
		} else {
			jsonBytes, err := json.MarshalIndent(cached, "", "  ")
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(jsonBytes))
		}
	},
}
