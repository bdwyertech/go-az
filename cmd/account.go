//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
		accountSetUserCmd,
		accountShowCmd,
		accountShowUserCmd,
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
	RunE: func(cmd *cobra.Command, args []string) error {
		subs, err := az.ListSubscriptionsCLI(cmd.Context(), false)
		if err != nil {
			return err
		}

		var sub interface{}
		subName, subId := viper.GetString("name"), viper.GetString("subscription")
		if subName != "" || subId != "" {
			for _, s := range subs {
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
				return errors.New("unable to find matching subscription")
			}
		} else {
			for _, s := range subs {
				if s.IsDefault {
					sub = s
					break
				}
			}
		}

		jsonBytes, err := json.MarshalIndent(sub, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonBytes))
		return nil
	},
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "Get a list of subscriptions for the logged in account.",
	// List All Subscriptions
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := az.ListSubscriptionsCLI(cmd.Context(), viper.GetBool("refresh"))
		if err != nil {
			return err
		}
		jsonBytes, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonBytes))
		return nil
	},
}

var accountGetAccessTokenCmd = &cobra.Command{
	Use:   "get-access-token",
	Short: "Get a token for utilities to access Azure.",
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := az.GetAccessToken(cmd.Context(), az.AccessTokenOptions{
			Resource:          viper.GetString("resource"),
			Scope:             viper.GetStringSlice("scope"),
			SubscriptionID:    viper.GetString("subscription"),
			Tenant:            viper.GetString("tenant"),
			Client:            viper.GetString("client"),
			PreferredUsername: accountHint(cmd),
		})
		if err != nil {
			return err
		}
		jsonBytes, err := json.MarshalIndent(u, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonBytes))
		return nil
	},
}

var accountCachedCmd = &cobra.Command{
	Use:   "cached",
	Short: "List cached accounts.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cached, err := az.GetCachedAccounts(cmd.Context())
		if err != nil {
			return err
		}
		if len(cached) == 0 {
			fmt.Println("[]")
			return nil
		}
		jsonBytes, err := json.MarshalIndent(cached, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonBytes))
		return nil
	},
}

// accountShowUserCmd mirrors the verb-first naming of the Microsoft Azure CLI's
// own `az account` subcommands. Microsoft reports the signed-in identity as the
// `user` block inside `az account show`; because this tool keeps several
// identities at once, that single field is not enough to describe the cache.
var accountShowUserCmd = &cobra.Command{
	Use:   "show-user",
	Short: "Show the active user and every cached user.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cached, err := az.GetCachedAccounts(cmd.Context())
		if err != nil {
			return err
		}

		report, err := az.BuildAccountReport(cmd.Context(), cached)
		if err != nil {
			return err
		}

		// A missing or stale pointer is a warning, not a failure: the remaining
		// selection precedence still yields a usable account.
		if report.ActiveHomeAccountID == "" {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "warning: no active user is recorded; run 'az account set-user'")
		} else if !report.ActiveIsCached {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: active user %q is no longer in the token cache\n", report.ActiveUsername)
		}

		jsonBytes, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(jsonBytes))
		return nil
	},
}

var accountSetUserCmd = &cobra.Command{
	Use:   "set-user",
	Short: "Record the active user without acquiring a token.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hint := accountHint(cmd)
		if len(args) == 1 {
			hint = args[0]
		}

		cached, err := az.GetCachedAccounts(cmd.Context())
		if err != nil {
			return err
		}

		selected, err := az.SetActiveAccount(cmd.Context(), cached, hint)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "active user is now %s\n", selected.PreferredUsername)
		return nil
	},
}
