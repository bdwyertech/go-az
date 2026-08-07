package az

import (
	"os"
	"strings"
)

// Account Hint environment variables, in precedence order. GO_AZ_USERNAME is
// go-az specific so a user can steer this tool without disturbing the real
// Azure CLI, which reads AZURE_USERNAME.
const (
	EnvAccountHint      = "GO_AZ_USERNAME"
	EnvAzureAccountHint = "AZURE_USERNAME"
)

// ResolveAccountHint returns the Account Hint for one invocation. The flag value
// wins; otherwise GO_AZ_USERNAME is consulted before AZURE_USERNAME. Values are
// trimmed, so a whitespace-only source is treated as absent.
func ResolveAccountHint(flag string) string {
	if h := strings.TrimSpace(flag); h != "" {
		return h
	}
	for _, k := range []string{EnvAccountHint, EnvAzureAccountHint} {
		if h := strings.TrimSpace(os.Getenv(k)); h != "" {
			return h
		}
	}
	return ""
}
