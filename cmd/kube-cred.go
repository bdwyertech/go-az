//
// Go AZ - Kubernetes Credential Provider
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
	rootCmd.AddCommand(kubeCredCmd)
	kubeCredCmd.Flags().StringP("name", "n", "", "Name of subscription.")
	kubeCredCmd.Flags().StringP("subscription", "s", "", "ID of subscription.")
	kubeCredCmd.Flags().StringP("resource", "", "", "Azure resource endpoints in AAD v1.0.")
	kubeCredCmd.Flags().StringSliceP("scope", "", []string{}, "Space-separated AAD scopes in AAD v2.0. Default to Azure Resource Manager.")
	kubeCredCmd.Flags().StringP("tenant", "t", "", "Tenant ID for which the token is acquired. Only available for user and service principal account, not for MSI or Cloud Shell account.")
	kubeCredCmd.Flags().StringP("client", "c", "", "Client Application ID for which the token is acquired.")
}

var kubeCredCmd = &cobra.Command{
	Use:   "kube-cred",
	Short: "Get a token for accessing Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := az.GetKubeCred(cmd.Context(), az.AccessTokenOptions{
			Resource:       viper.GetString("resource"),
			Scope:          viper.GetStringSlice("scope"),
			SubscriptionID: viper.GetString("subscription"),
			Tenant:         viper.GetString("tenant"),
			Client:         viper.GetString("client"),
		})
		if err != nil {
			log.Fatal(err)
		}
		jsonBytes, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(jsonBytes))
	},
}
