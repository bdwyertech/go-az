//
// Go AZ
//
// Copyright © 2022 Brian Dwyer - Intelligent Digital Services. All rights reserved.
//

package cmd

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bdwyertech/go-az/pkg/az"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	loginCmd.Flags().StringP("scope", "", "", "Used in the /authorize request. It can cover only one static resource.")
	loginCmd.Flags().BoolP("force", "", false, "Always prompt for a fresh sign-in, ignoring any cached token.")
	loginCmd.Flags().StringP("tenant", "t", "", "Tenant ID for which the token is acquired. Only available for user and service principal account, not for MSI or Cloud Shell account.")
	// loginCmd.Flags().BoolP("allow-no-subscriptions", "", false, "Support access tenants without subscriptions.")
	// loginCmd.Flags().BoolP("use-device-code", "", false, "Use CLI's old authentication flow based on device code.")
	// loginCmd.Flags().StringP("federated-token", "", "", "Federated token that can be used for OIDC token exchange.")
	// loginCmd.Flags().StringP("service-principal", "", "", "The credential representing a service principal.")
	// loginCmd.Flags().StringP("username", "u", "", "User name, service principal, or managed service identity ID.")
	// loginCmd.Flags().StringP("password", "p", "", "Credentials like user password, or for a service principal, provide client secret or a pem file with key and public certificate. Will")

	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Azure.",
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		opts := az.AccessTokenOptions{
			Tenant:            viper.GetString("tenant"),
			PreferredUsername: accountHint(cmd),
			ForceInteractive:  force,
		}
		if scope := viper.GetString("scope"); scope != "" {
			opts.Scope = append(opts.Scope, scope)
		}

		tok, err := az.GetAccessToken(cmd.Context(), opts)
		if err != nil {
			// A failed login must change nothing, so the active user is only
			// recorded after the exchange succeeds.
			return err
		}
		if err := az.RecordActiveAccount(cmd.Context(), tok.Account()); err != nil {
			log.Warnf("unable to record the active user: %v", err)
		}
		// Scope the refresh to the identity that just authenticated rather than
		// the raw hint: the hint may have been empty, and the account returned by
		// the exchange is the authoritative answer to "who is this profile for".
		//
		// A partial enumeration is still worth printing, so a listing error is
		// reported without discarding the subscriptions that did come back.
		s, err := az.ListSubscriptionsCLI(cmd.Context(), true, tok.Account().PreferredUsername)
		if err != nil {
			log.Warnf("unable to enumerate every subscription: %v", err)
		}
		jsonBytes, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonBytes))
		return nil
	},
}
