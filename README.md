

All we really have to emulate is the following:

az account get-access-token -o json --resource https://management.core.windows.net/


#### TestKitchen AZ CLI Credential Source
https://github.com/test-kitchen/kitchen-azurerm/blob/c9fc65b6ca554d0c8e833f83e55150c5af7cabe3/lib/kitchen/driver/azure_credentials.rb#L115-L121

#### Azure SDK ShellOut
https://github.com/Azure/azure-sdk-for-ruby/blob/9d0fd011848f829bef2f0987e3d2db22fd179106/runtime/ms_rest_azure/lib/ms_rest_azure/credentials/azure_cli_token_provider.rb#L70-L77

We can probably set PATHEXT to go, or ensure we are the first `az` on the PATH to prevent collisions with the actual Azure CLI

