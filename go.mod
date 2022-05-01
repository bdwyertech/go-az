module github.com/bdwyertech/go-az

go 1.16

// replace github.com/AzureAD/microsoft-authentication-library-for-go => github.com/bdwyertech/microsoft-authentication-library-for-go v0.2.1-0.20220116010247-e0ef7800a7b8

// access-via-refresh
replace github.com/AzureAD/microsoft-authentication-library-for-go => github.com/bdwyertech/microsoft-authentication-library-for-go v0.2.1-0.20220119214522-7c10a8cb4b96

require (
	github.com/Azure/azure-sdk-for-go v63.4.0+incompatible
	github.com/Azure/azure-sdk-for-go/sdk/azcore v0.21.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription v0.2.0
	github.com/Azure/go-autorest/autorest v0.11.24
	github.com/Azure/go-autorest/autorest/adal v0.9.18
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.5
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v0.4.0
	github.com/google/uuid v1.3.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.3.0
	github.com/spf13/viper v1.10.1
)
