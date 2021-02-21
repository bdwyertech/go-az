package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// Azure CLI Default Client ID & Tenant
// https://github.com/Azure/azure-cli/blob/7bcf85939f00fbfc1961b9a82f0584e17e33e577/src/azure-cli-core/azure/cli/core/_profile.py#L69
const AZ_CLIENT_ID = "04b07795-8ddb-461a-bbee-02f9e1bf7b46"

//
// // Authority: https://login.microsoftonline.com/common
// // Specific Tenant: https://login.microsoftonline.com/my-tenant-id
// const AZ_COMMON_TENANT = "common"
//
// var (
// 	publicClientApp *msal.PublicClientApplication
// 	scopes          = []string{"user.read"}
// )

func main() {
	pubClient, err := public.New(AZ_CLIENT_ID)

	port, err := getFreePort()
	if err != nil {
		log.Fatal(err)
	}
	redirectUrl := fmt.Sprintf("http://localhost:%v", port)
	token, err := pubClient.AcquireTokenInteractive(context.Background(), []string{"https://management.core.windows.net/.default"}, public.WithRedirectURI(redirectUrl))
	if err != nil {
		log.Fatal(err)
	}

	jsonBytes, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(jsonBytes))
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
