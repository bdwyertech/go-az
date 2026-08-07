

# go-az

A single-binary Go replacement for the parts of the Azure CLI that other tools
actually shell out to. It signs in interactively, keeps its own MSAL token
cache, and writes an Azure CLI compatible profile so existing tooling keeps
working.

The original goal was to emulate this one call:

```console
az account get-access-token -o json --resource https://management.core.windows.net/
```


## Commands

| Command | Purpose |
| --- | --- |
| `az login` | Sign in interactively and print the subscriptions that were found. |
| `az account show` | Print the default subscription from the profile. |
| `az account list` | Print every known subscription. |
| `az account get-access-token` | Print an access token for a resource or scope. |
| `az account cached` | Print the accounts present in the token cache. |
| `az account show-user` | Print the active user and every cached user. |
| `az account set-user` | Record which cached user later commands should use. |
| `az ad signed-in-user` | Print the signed-in user from Microsoft Graph. |
| `az kube-cred` | Print a Kubernetes `ExecCredential` for `kubelogin`-style use. |
| `az organizations` | Print organization details from Microsoft Graph. |
| `az tenants` | Print the tenants the signed-in user can reach. |
| `az version` | Print the version, or a Terraform plugin handshake when asked. |

Global flags: `--output/-o` (only `json`), `--debug`, and
`--preferred-username`.

## Environment variables

| Variable | Effect |
| --- | --- |
| `GO_AZ_USERNAME` | Account Hint. Selects which cached user to act as. |
| `AZURE_USERNAME` | Account Hint, consulted only when `GO_AZ_USERNAME` is unset. |
| `GO_AZ_DEVICECODE` | Any non-empty value switches interactive login to the device code flow. |
| `KUBERNETES_EXEC_INFO` | Set by `kubectl`; consumed by `az kube-cred`. |
| `TF_PLUGIN_MAGIC_COOKIE` | Set by Terraform; makes `az version` emit a plugin handshake instead of a version. |

## Files

All three files live in the credential directory, `~/.azure` by default.

| File | Contents |
| --- | --- |
| `~/.azure/go_msal_token_cache.json` | Token Cache. MSAL accounts, refresh tokens, and access tokens. |
| `~/.azure/azureProfile.json` | Profile. Azure CLI compatible subscription list, so other tools can read it. |
| `~/.azure/go_az_state.json` | State File. Records the active user only; kept separate so the Profile stays Azure CLI compatible. |

Writes are atomic and guarded by an advisory file lock, so a concurrent Azure
CLI or a second `go-az` cannot observe a half-written file.

## Selection Precedence

Several users can be signed in at once. Every command resolves exactly one of
them from a single snapshot of the cache, taking the first rule that matches:

1. **Account Hint.** `--preferred-username`, then `GO_AZ_USERNAME`, then
   `AZURE_USERNAME`. The hint is matched case-insensitively against the
   username, the local account ID, and the home account ID. A hint that matches
   nothing is an error rather than a silent fallback.
2. **Active user.** The home account ID recorded by `az account set-user`.
3. **Tenant.** `--tenant`, when exactly one cached user calls that tenant home.
4. **Sole user.** When only one user is cached.

If none apply the command fails as ambiguous instead of guessing, because
guessing is what produced tokens for the wrong identity.

## References

#### TestKitchen AZ CLI Credential Source
https://github.com/test-kitchen/kitchen-azurerm/blob/c9fc65b6ca554d0c8e833f83e55150c5af7cabe3/lib/kitchen/driver/azure_credentials.rb#L115-L121

#### Azure SDK ShellOut
https://github.com/Azure/azure-sdk-for-ruby/blob/9d0fd011848f829bef2f0987e3d2db22fd179106/runtime/ms_rest_azure/lib/ms_rest_azure/credentials/azure_cli_token_provider.rb#L70-L77

We can probably set PATHEXT to go, or ensure we are the first `az` on the PATH to prevent collisions with the actual Azure CLI

