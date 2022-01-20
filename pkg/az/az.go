package az

import "github.com/Azure/go-autorest/autorest/azure"

const (
	// Azure CLI Default Client ID & Tenant
	// https://github.com/Azure/azure-cli/blob/7bcf85939f00fbfc1961b9a82f0584e17e33e577/src/azure-cli-core/azure/cli/core/_profile.py#L69
	AZ_CLIENT_ID = "04b07795-8ddb-461a-bbee-02f9e1bf7b46"
)

var (
	microsoftAuthorityHost = azure.PublicCloud.ActiveDirectoryEndpoint // "https://login.microsoftonline.com/"
	// organizationsAuthority = microsoftAuthorityHost + "organizations"
	// commonAuthority        = microsoftAuthorityHost + "common"
)
