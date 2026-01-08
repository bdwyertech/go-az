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
	loginCmd.Flags().StringP("tenant", "t", "", "Tenant ID for which the token is acquired. Only available for user and service principal account, not for MSI or Cloud Shell account.")
	loginCmd.Flags().BoolP("interactive", "", false, "Force interactive login")
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
	Run: func(cmd *cobra.Command, args []string) {
		opts := new(az.TokenOptions)
		opts.ForceInteractive = viper.GetBool("interactive")
		opts.TokenRequestOptions.TenantID = viper.GetString("tenant")

		if scope := viper.GetString("scope"); scope != "" {
			opts.Scopes = append(opts.Scopes, scope)
		}

		token, err := az.GetAccessToken(cmd.Context(), opts)
		if err != nil {
			log.Fatal(err)
		}

		log.Debugln("Authenticated as: ", token.Username)

		// Build profile with authenticated user to ensure correct user association
		if err := az.BuildProfileWithUser(token.Username); err != nil {
			log.Fatal(err)
		}

		s := az.ListSubscriptionsCLI(false) // false because we just built the profile
		jsonBytes, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}
